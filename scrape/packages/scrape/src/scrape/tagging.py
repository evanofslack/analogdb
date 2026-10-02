import json
import re
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional, Sequence, Set

import inflect
from openai import BadRequestError, OpenAI, OpenAIError

from .models import Caption, Keyword

TAGGER_VERSION = "v1"
TEXT_ONLY_SUFFIX = "-text"
MAX_TAGS = 15
TEXT_LIMIT = 1000
MAX_WEIGHT = 1.0
MIN_WEIGHT = 0.5

SYSTEM_PROMPT = """You describe photos from an analog photography forum so people can search for them.

Return JSON with a caption and tags.

Caption: one plain sentence describing what is visible in the photo. Never mention the camera, film or lens.

Tags: 5 to 15 tags, most important first. Lowercase, singular, mostly single words:
- subjects: woman, dog, car, building, tree
- setting: beach, street, forest, mountain, kitchen
- time and weather: sunset, night, fog, snow, rain
- notable colors or light: backlit, neon, golden hour, shadow
- places named in the title or description, which may be several words: new york, tokyo, lake district

Only tag what is visible in the photo, plus places the author names. Never tag a camera, lens, film, brand or format, and never use the words photo, film, analog, picture or image.

Return JSON: {"caption": "...", "tags": ["...", "..."]}"""

TEXT_ONLY_NOTE = "The photo is not available. Tag only what the title and description name, and leave the caption empty."

RESPONSE_SCHEMA = {
    "type": "object",
    "properties": {
        "caption": {"type": "string"},
        "tags": {"type": "array", "items": {"type": "string"}, "maxItems": MAX_TAGS},
    },
    "required": ["caption", "tags"],
    "additionalProperties": False,
}

SINGULAR_KEEP = {
    "alps",
    "angeles",
    "athens",
    "atlas",
    "canvas",
    "christmas",
    "clothes",
    "glasses",
    "jeans",
    "news",
    "people",
    "series",
    "shorts",
    "species",
    "stairs",
    "sunglasses",
    "texas",
    "vegas",
    "wales",
}
SINGULAR_SKIP_ENDINGS = ("ss", "us", "is", "ics")

_inflect = inflect.engine()


class TaggingError(Exception):
    pass


@dataclass
class ImageTags:
    caption: Optional[str]
    tags: List[str]
    raw: Dict
    model: str
    version: str

    @property
    def text_only(self) -> bool:
        return self.version.endswith(TEXT_ONLY_SUFFIX)

    def keywords(self) -> List[Keyword]:
        return tag_keywords(self.tags)

    def post_caption(self) -> Caption:
        return Caption(
            caption=self.caption, model=self.model, version=self.version, raw=self.raw
        )


@dataclass
class TagInput:
    image_url: str
    title: str
    description: Optional[str] = None


def tag_keywords(tags: Sequence[str]) -> List[Keyword]:
    """Weights fall linearly from 1.0 for the first tag to 0.5 for the last."""
    if len(tags) == 1:
        return [Keyword(word=tags[0], weight=MAX_WEIGHT)]
    step = (MAX_WEIGHT - MIN_WEIGHT) / max(len(tags) - 1, 1)
    return [
        Keyword(word=tag, weight=round(MAX_WEIGHT - i * step, 4))
        for i, tag in enumerate(tags)
    ]


def clean_tag(tag: str) -> str:
    tag = re.sub(r"[^\w\s-]|_", " ", tag.lower())
    words = [w.strip("-") for w in tag.split()]
    return " ".join(w for w in words if w)


def singular(tag: str) -> str:
    *head, last = tag.split(" ")
    if (
        len(last) <= 3
        or last in SINGULAR_KEEP
        or last.endswith(SINGULAR_SKIP_ENDINGS)
        or not last.isalpha()
    ):
        return tag
    single = _inflect.singular_noun(last)
    if not single:
        return tag
    return " ".join([*head, single])


def named_in(text: str) -> Set[str]:
    """Capitalized words and runs of words in the text, lowercased: the place
    names the author wrote, which are never singularized."""
    names: Set[str] = set()
    for run in re.findall(r"[A-Z][\w'-]*(?:\s+[A-Z][\w'-]*)*", text):
        words = clean_tag(run).split()
        for size in range(1, len(words) + 1):
            for start in range(len(words) - size + 1):
                names.add(" ".join(words[start : start + size]))
    return names


def normalize_tags(
    tags: Iterable[Any], blocked: Set[str], named: Optional[Set[str]] = None
) -> List[str]:
    named = named or set()
    out: List[str] = []
    for tag in tags:
        if not isinstance(tag, str):
            continue
        tag = clean_tag(tag)
        if not tag or tag in blocked:
            continue
        if tag not in named:
            tag = singular(tag)
        if tag in blocked or tag in out:
            continue
        out.append(tag)
        if len(out) == MAX_TAGS:
            break
    return out


def catalog_words(cameras: List[Dict], films: List[Dict]) -> Set[str]:
    """Camera and film makes and names from the catalog json, whole names only,
    so a film called "gold 200" blocks "gold 200" but not "gold"."""
    names: Set[str] = set()
    for c in cameras:
        make, model = c.get("make") or "", c.get("model") or ""
        names.update([make, model, f"{make} {model}"])
        names.update(c.get("aliases") or [])
    for f in films:
        make, type = f.get("make") or "", f.get("type") or ""
        names.update([make, type, f"{make} {type}"])
        names.update(f.get("aliases") or [])
    words = {clean_tag(n) for n in names}
    return {w for w in words if len(w) > 1 and not w.isdigit()}


def load_stoplist(path: str) -> Set[str]:
    file = Path(path)
    if not file.exists():
        return set()
    lines = file.read_text(encoding="utf-8").splitlines()
    return {clean_tag(line) for line in lines if clean_tag(line)}


def medium_url(images: Sequence[Any]) -> str:
    """The medium CloudFront image of a post, for scrape and API images alike."""
    for image in images:
        if image.resolution == "medium" and image.url:
            return image.url
    raise ValueError("no medium image")


def post_text(title: str, description: Optional[str]) -> str:
    def clean(text: Optional[str]) -> str:
        return " ".join((text or "").split())

    text = f"title: {clean(title)}"
    if clean(description):
        text += f"\ndescription: {clean(description)}"
    return text[:TEXT_LIMIT]


def parse_tags(content: Optional[str]) -> Dict:
    """The model's JSON object, or ValueError for refusals, prose and results
    without tags."""
    if content is None or not content.strip():
        raise ValueError("empty response")
    content = content.strip()
    if content.startswith("```"):
        content = re.sub(r"^```(?:json)?\s*", "", content)
        content = re.sub(r"\s*```$", "", content)
    try:
        data = json.loads(content)
    except json.JSONDecodeError as e:
        raise ValueError(f"not json: {content[:100]}") from e
    if not isinstance(data, dict):
        raise ValueError("not a json object")
    tags = data.get("tags")
    if not isinstance(tags, list) or not tags:
        raise ValueError("no tags")
    return data


def caption_text(data: Dict) -> Optional[str]:
    caption = data.get("caption")
    if not isinstance(caption, str):
        return None
    return " ".join(caption.split()) or None


class ImageTagger:
    """Caption and tag one image per request with a vision model, then
    normalize the tags in code."""

    MAX_ATTEMPTS = 2

    def __init__(
        self,
        openai: OpenAI,
        model: str,
        catalog_words: Set[str],
        stoplist: Optional[Set[str]] = None,
    ):
        self.openai = openai
        self.model = model
        self.blocked = set(catalog_words) | set(stoplist or ())
        self.use_schema = True

    def tag(self, image_url: str, title: str, description: Optional[str]) -> ImageTags:
        text = post_text(title, description)
        named = named_in(f"{title}\n{description or ''}")
        image = [
            {"type": "image_url", "image_url": {"url": image_url}},
            {"type": "text", "text": text},
        ]
        found = self._attempt(image, named)
        if found is not None:
            data, tags = found
            return ImageTags(
                caption=caption_text(data),
                tags=tags,
                raw=data,
                model=self.model,
                version=TAGGER_VERSION,
            )

        found = self._attempt(
            [{"type": "text", "text": f"{text}\n\n{TEXT_ONLY_NOTE}"}], named
        )
        if found is None:
            raise TaggingError("no tags from the image or the title")
        data, tags = found
        return ImageTags(
            caption=None,
            tags=tags,
            raw=data,
            model=self.model,
            version=TAGGER_VERSION + TEXT_ONLY_SUFFIX,
        )

    def tag_all(
        self, inputs: Sequence[TagInput], concurrency: int = 8
    ) -> List[ImageTags | Exception]:
        """Tag concurrently. Each result is the tags or the exception raised."""

        def run(i: TagInput) -> ImageTags | Exception:
            try:
                return self.tag(i.image_url, i.title, i.description)
            except Exception as e:
                return e

        with ThreadPoolExecutor(max_workers=max(concurrency, 1)) as pool:
            return list(pool.map(run, inputs))

    def _attempt(self, content: List[Dict], named: Set[str]):
        for _ in range(self.MAX_ATTEMPTS):
            try:
                data = parse_tags(self._query(content))
            except BadRequestError:
                self.use_schema = False
                continue
            except (OpenAIError, ValueError):
                continue
            tags = normalize_tags(data["tags"], self.blocked, named)
            if tags:
                return data, tags
        return None

    def _query(self, content: List[Dict]) -> Optional[str]:
        if self.use_schema:
            response_format = {
                "type": "json_schema",
                "json_schema": {
                    "name": "image_tags",
                    "strict": True,
                    "schema": RESPONSE_SCHEMA,
                },
            }
        else:
            response_format = {"type": "json_object"}
        resp = self.openai.chat.completions.create(
            extra_headers={"HTTP-Referer": "analogdb.com", "X-Title": "analogdb.com"},
            model=self.model,
            messages=[
                {"role": "system", "content": SYSTEM_PROMPT},
                {"role": "user", "content": content},
            ],
            temperature=0,
            response_format=response_format,
        )
        message = resp.choices[0].message
        if getattr(message, "refusal", None):
            raise ValueError(f"refused: {message.refusal}")
        return message.content
