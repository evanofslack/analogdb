import json
import re
from typing import Dict, List, Optional

from analogdb.models import Camera, Film
from openai import BadRequestError, OpenAI, OpenAIError

from .catalog_match import CatalogAlias, CatalogMatcher
from .models import MatchResult, MetadataPost

EXTRACTOR_VERSION = "2026-10-a"

EXTRA_CAMERA_MAKES = [
    "Lomography",
    "Plaubel",
    "Agfa",
    "Argus",
    "Fed",
    "Lubitel",
    "Smena",
]
EXTRA_FILM_MAKES = ["Svema", "Kodak Professional", "Fomapan"]

SYSTEM_PROMPT = """You read posts from an analog photography forum and copy out the cameras, films and lenses each post says its photos were shot with. You transcribe. You don't correct.

Rules:
- Copy names as written. Don't correct them to a product you know: "Nikon F-3" stays model "F-3", "Portra400" stays type "Portra400", "Contax G2" stays "G2".
- List every camera, film and lens in order of mention: title, then description, then the photographer's comments.
- make: the brand as written. When it isn't written, fill it only if the name clearly implies it (Portra → Kodak, HP5 → Ilford, RB67 → Mamiya, AE-1 → Canon, Superia → Fujifilm). Otherwise null.
- model or type: the name after the brand, without the brand.
- box_speed: the film's rated ISO when written or part of the name (Portra 400 → 400). shot_at: a push or pull speed (HP5 @ 1600, pushed to 800). Otherwise null.
- focal_length: mm as an integer, the first number of a range (24-70mm → 24). aperture: like "f/2.8".
- Lens names are not cameras (Zeiss, Nikkor, Summicron, Sekor, Voigtlander 35mm). Developers (D76, Rodinal, HC-110), scanners (Noritsu, Epson), papers and chemicals are not films.
- Only gear the photos were shot with counts, not gear asked about, wished for or compared.
- text: the exact words you copied from.
- Empty lists when the post names nothing.

Known camera makes: {camera_makes}
Known film makes: {film_makes}

Return JSON: {{"posts": [{{"post_id": 1, "cameras": [], "films": [], "lenses": []}}]}}, one object per post, in order."""

EXAMPLE_INPUT = """post_id: 1
title: From my Balcony [Canon A-1 / FD 35-105mm F3.5 / Fuji Superia 400]

post_id: 2
title: I dreamed I was walking home (RB67, 50mm, Delta 400, Rodinal)

post_id: 3
title: Lomo CN 100 || Canon Eos 5 || Lomo Neptune 35mm f/3.5

post_id: 4
title: Sliding through the stars [Pentax LX + Provia 100F]

post_id: 5
title: Harbour lights, HP5+ pushed to 1600
op comments: Shot this on my dad's old Nikon N90s. Thinking about getting a Leica M6 next.

post_id: 6
title: Sunday morning at the market
op comments: Thanks! Developed at home in D76."""

EXAMPLE_OUTPUT = {
    "posts": [
        {
            "post_id": 1,
            "cameras": [{"text": "Canon A-1", "make": "Canon", "model": "A-1"}],
            "films": [
                {
                    "text": "Fuji Superia 400",
                    "make": "Fuji",
                    "type": "Superia 400",
                    "box_speed": 400,
                    "shot_at": None,
                }
            ],
            "lenses": [
                {"text": "FD 35-105mm F3.5", "focal_length": 35, "aperture": "f/3.5"}
            ],
        },
        {
            "post_id": 2,
            "cameras": [{"text": "RB67", "make": "Mamiya", "model": "RB67"}],
            "films": [
                {
                    "text": "Delta 400",
                    "make": "Ilford",
                    "type": "Delta 400",
                    "box_speed": 400,
                    "shot_at": None,
                }
            ],
            "lenses": [{"text": "50mm", "focal_length": 50, "aperture": None}],
        },
        {
            "post_id": 3,
            "cameras": [{"text": "Canon Eos 5", "make": "Canon", "model": "Eos 5"}],
            "films": [
                {
                    "text": "Lomo CN 100",
                    "make": "Lomo",
                    "type": "CN 100",
                    "box_speed": 100,
                    "shot_at": None,
                }
            ],
            "lenses": [
                {
                    "text": "Lomo Neptune 35mm f/3.5",
                    "focal_length": 35,
                    "aperture": "f/3.5",
                }
            ],
        },
        {
            "post_id": 4,
            "cameras": [{"text": "Pentax LX", "make": "Pentax", "model": "LX"}],
            "films": [
                {
                    "text": "Provia 100F",
                    "make": "Fujifilm",
                    "type": "Provia 100F",
                    "box_speed": 100,
                    "shot_at": None,
                }
            ],
            "lenses": [],
        },
        {
            "post_id": 5,
            "cameras": [{"text": "Nikon N90s", "make": "Nikon", "model": "N90s"}],
            "films": [
                {
                    "text": "HP5+ pushed to 1600",
                    "make": "Ilford",
                    "type": "HP5+",
                    "box_speed": 400,
                    "shot_at": 1600,
                }
            ],
            "lenses": [],
        },
        {"post_id": 6, "cameras": [], "films": [], "lenses": []},
    ]
}

_NULLABLE_STRING = {"type": ["string", "null"]}
_NULLABLE_INT = {"type": ["integer", "null"]}


def _object(properties: Dict) -> Dict:
    return {
        "type": "object",
        "properties": properties,
        "required": list(properties),
        "additionalProperties": False,
    }


RESPONSE_SCHEMA = _object(
    {
        "posts": {
            "type": "array",
            "items": _object(
                {
                    "post_id": {"type": "integer"},
                    "cameras": {
                        "type": "array",
                        "items": _object(
                            {
                                "text": {"type": "string"},
                                "make": _NULLABLE_STRING,
                                "model": _NULLABLE_STRING,
                            }
                        ),
                    },
                    "films": {
                        "type": "array",
                        "items": _object(
                            {
                                "text": {"type": "string"},
                                "make": _NULLABLE_STRING,
                                "type": _NULLABLE_STRING,
                                "box_speed": _NULLABLE_INT,
                                "shot_at": _NULLABLE_INT,
                            }
                        ),
                    },
                    "lenses": {
                        "type": "array",
                        "items": _object(
                            {
                                "text": {"type": "string"},
                                "focal_length": _NULLABLE_INT,
                                "aperture": _NULLABLE_STRING,
                            }
                        ),
                    },
                }
            ),
        }
    }
)


def post_text(post: MetadataPost, limit: int = 1980) -> str:
    """Title, description and OP comments as the LLM sees them, one per line."""

    def clean(text: Optional[str]) -> str:
        return " ".join((text or "").split())

    lines = [f"title: {clean(post.title)}"]
    if clean(post.description):
        lines.append(f"description: {clean(post.description)}")
    text = "\n".join(lines)
    comments = " / ".join(clean(c) for c in post.op_comments if clean(c))
    room = limit - len(text) - len("\nop comments: ")
    if comments and room > 0:
        text += f"\nop comments: {comments[:room]}"
    return text[:limit]


def post_input(post_id: int, post: MetadataPost) -> str:
    return f"post_id: {post_id}\n{post_text(post)}"


class Transcriber:
    """Stage 1: the LLM copies camera, film and lens mentions as written."""

    MAX_ATTEMPTS = 2

    def __init__(self, openai: OpenAI, llm_model: str, batch_size: int = 20):
        self.openai = openai
        self.llm_model = llm_model
        self.batch_size = batch_size
        self.use_schema = True

    def transcribe(
        self,
        posts: List[MetadataPost],
        camera_makes: List[str],
        film_makes: List[str],
    ) -> List[Optional[Dict]]:
        """One dict per post (cameras, films, lenses), None where the LLM failed."""
        system = SYSTEM_PROMPT.format(
            camera_makes=", ".join(sorted(set(camera_makes) | set(EXTRA_CAMERA_MAKES))),
            film_makes=", ".join(sorted(set(film_makes) | set(EXTRA_FILM_MAKES))),
        )
        results: List[Optional[Dict]] = [None] * len(posts)
        for start in range(0, len(posts), self.batch_size):
            chunk = posts[start : start + self.batch_size]
            ids = list(range(1, len(chunk) + 1))
            prompt = "\n\n".join(post_input(i, p) for i, p in zip(ids, chunk))
            by_id = self._query_chunk(system, prompt, ids)
            for i in ids:
                results[start + i - 1] = by_id.get(i)
        return results

    def _query_chunk(self, system: str, prompt: str, ids: List[int]) -> Dict[int, Dict]:
        for _ in range(self.MAX_ATTEMPTS):
            try:
                content = self._query(system, prompt)
                return parse_transcripts(content, ids)
            except BadRequestError:
                if not self.use_schema:
                    continue
                self.use_schema = False
            except (OpenAIError, json.JSONDecodeError, ValueError):
                continue
        return {}

    def _query(self, system: str, prompt: str) -> Optional[str]:
        if self.use_schema:
            response_format = {
                "type": "json_schema",
                "json_schema": {
                    "name": "transcripts",
                    "strict": True,
                    "schema": RESPONSE_SCHEMA,
                },
            }
        else:
            response_format = {"type": "json_object"}
        resp = self.openai.chat.completions.create(
            extra_headers={"HTTP-Referer": "analogdb.com", "X-Title": "analogdb.com"},
            model=self.llm_model,
            messages=[
                {"role": "system", "content": system},
                {"role": "user", "content": EXAMPLE_INPUT},
                {"role": "assistant", "content": json.dumps(EXAMPLE_OUTPUT)},
                {"role": "user", "content": prompt},
            ],
            temperature=0,
            response_format=response_format,
        )
        return resp.choices[0].message.content


def parse_transcripts(content: Optional[str], ids: List[int]) -> Dict[int, Dict]:
    if content is None:
        raise ValueError("empty response")
    content = content.strip()
    if content.startswith("```"):
        content = re.sub(r"^```(?:json)?\s*", "", content)
        content = re.sub(r"\s*```$", "", content)
    data = json.loads(content)
    if isinstance(data, dict):
        data = data.get(
            "posts", next((v for v in data.values() if isinstance(v, list)), None)
        )
    if not isinstance(data, list):
        raise ValueError("no posts list")
    by_id: Dict[int, Dict] = {}
    for item in data:
        if not isinstance(item, dict):
            continue
        raw_id = item.get("post_id")
        if isinstance(raw_id, bool):
            continue
        try:
            post_id = int(raw_id)
        except (TypeError, ValueError):
            continue
        if post_id not in ids or post_id in by_id:
            continue
        by_id[post_id] = {
            "cameras": [c for c in item.get("cameras") or [] if isinstance(c, dict)],
            "films": [f for f in item.get("films") or [] if isinstance(f, dict)],
            "lenses": [
                lens for lens in item.get("lenses") or [] if isinstance(lens, dict)
            ],
        }
    return by_id


class MetadataExtractor:
    """Transcribe with the LLM, then match to the catalog in code."""

    def __init__(self, openai: OpenAI, llm_model: str, batch_size: int = 20):
        self.transcriber = Transcriber(openai, llm_model, batch_size)

    def extract(
        self,
        posts: List[MetadataPost],
        cameras: List[Camera],
        films: List[Film],
        aliases: Optional[List[CatalogAlias]] = None,
    ) -> List[MatchResult]:
        raws = self.transcriber.transcribe(
            posts,
            sorted({c.make.lower() for c in cameras}),
            sorted({f.make.lower() for f in films}),
        )
        matcher = CatalogMatcher(cameras, films, aliases)
        return [matcher.match(raw, post) for raw, post in zip(raws, posts, strict=True)]
