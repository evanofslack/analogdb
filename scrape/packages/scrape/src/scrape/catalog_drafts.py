"""Draft new camera and film catalog entries from the unmatched mentions.

candidates ranks the mentions the catalog doesn't have. A triage file, written by
hand, says which become entries and which are aliases. find_source looks each new
entry up on camera-wiki and Wikipedia, draft_prompt asks an LLM to write the
description from that source in the catalog's style, and apply_drafts adds the
reviewed entries and aliases to the catalog JSON.
"""

import copy
import difflib
import json
import re
from collections import Counter, defaultdict
from dataclasses import dataclass, field
from html.parser import HTMLParser
from typing import Any, Callable, Dict, Iterable, List, Optional, Tuple

from .normalize import normalize_key

Entry = Dict[str, Any]
Fetch = Callable[[str, Dict[str, Any]], Dict[str, Any]]

CAMERA_WIKI = "https://camera-wiki.org/api.php"
CAMERA_WIKI_PAGE = "https://camera-wiki.org/wiki/"
WIKIPEDIA = "https://en.wikipedia.org/w/api.php"
WIKIPEDIA_PAGE = "https://en.wikipedia.org/wiki/"
SOURCE_CHARS = 6000
TARGET_WORDS = {"camera": 130, "film": 60}
WORD_RANGE = {"camera": (25, 230), "film": (20, 130)}
COLOR_TYPES = {"color", "bw"}
EXAMPLES = 3


def name_field(kind: str) -> str:
    return "model" if kind == "camera" else "type"


@dataclass
class Candidate:
    kind: str
    key: str
    posts: int
    make: Optional[str]
    name: Optional[str]
    speed: Optional[int]
    spellings: List[str]
    examples: List[int]
    nearest: List[str] = field(default_factory=list)


def candidates(
    extractions: Iterable[Dict[str, Any]],
    cameras: List[Entry],
    films: List[Entry],
    min_posts: int = 5,
) -> List[Candidate]:
    """Unmatched keys with at least min_posts posts, most posts first, with the
    closest catalog names to help decide between a new entry and an alias."""
    posts: Dict[Tuple[str, str], set] = defaultdict(set)
    spellings: Dict[Tuple[str, str], Counter] = defaultdict(Counter)
    makes: Dict[Tuple[str, str], Counter] = defaultdict(Counter)
    names: Dict[Tuple[str, str], Counter] = defaultdict(Counter)
    speeds: Dict[Tuple[str, str], Counter] = defaultdict(Counter)
    for e in extractions:
        for u in e.get("unmatched") or []:
            k = (u.get("kind"), u.get("key"))
            if not k[0] or not k[1]:
                continue
            posts[k].add(e["post_id"])
            spellings[k][u.get("raw") or k[1]] += 1
            makes[k][u.get("make")] += 1
            names[k][u.get(name_field(k[0]))] += 1
            if u.get("speed"):
                speeds[k][u["speed"]] += 1

    catalog = {
        "camera": [f"{c['make']} {c['model']}" for c in cameras],
        "film": [f"{f['make']} {f['type']}" for f in films],
    }
    out = []
    for k, ids in posts.items():
        if len(ids) < min_posts:
            continue
        kind, key = k
        make = makes[k].most_common(1)[0][0]
        name = names[k].most_common(1)[0][0]
        speed = speeds[k].most_common(1)[0][0] if speeds[k] else None
        out.append(
            Candidate(
                kind=kind,
                key=key,
                posts=len(ids),
                make=make,
                name=name,
                speed=speed,
                spellings=[s for s, _ in spellings[k].most_common(3)],
                examples=sorted(ids, reverse=True)[:4],
                nearest=difflib.get_close_matches(
                    f"{make or ''} {name or ''}".strip(), catalog[kind], n=3, cutoff=0.5
                ),
            )
        )
    return sorted(out, key=lambda c: (-c.posts, c.kind, c.key))


@dataclass
class Source:
    site: str
    title: str
    url: str
    text: str


class _Text(HTMLParser):
    """Visible text of a MediaWiki page, without scripts, styles and edit links."""

    SKIP = {"script", "style", "sup"}
    VOID = {"area", "br", "col", "hr", "img", "input", "link", "meta", "source", "wbr"}
    BLOCK = {"br", "h2", "h3", "li", "p", "tr"}

    def __init__(self):
        super().__init__()
        self.parts: List[str] = []
        self.depth = 0

    def handle_starttag(self, tag, attrs):
        if self.depth:
            if tag not in self.VOID:
                self.depth += 1
            return
        cls = dict(attrs).get("class") or ""
        if (tag in self.SKIP or "mw-editsection" in cls) and tag not in self.VOID:
            self.depth = 1
        elif tag in self.BLOCK:
            self.parts.append("\n")

    def handle_endtag(self, tag):
        if self.depth and tag not in self.VOID:
            self.depth -= 1

    def handle_data(self, data):
        if not self.depth:
            self.parts.append(data)


def html_text(page: str) -> str:
    parser = _Text()
    parser.feed(page)
    text = "".join(parser.parts)
    text = re.sub(r"[ \t\xa0]+", " ", text)
    text = re.sub(r"\s*\n\s*", "\n", text)
    return text.strip()


def title_fits(
    title: str, kind: str, name: str, speed: Optional[int], make: str = ""
) -> bool:
    """A search hit is about the entry: the title names the make (its first letters,
    so "Fujica" is a fujifilm) and the model, or for films the film's name without
    its speed ("Fujifilm Superia" for Superia 200)."""
    title_key, name_key = normalize_key(title), normalize_key(name)
    make_key = normalize_key(make)[:4]
    # Some pages leave the make out ("Bessa R"), fine for a name that's mostly letters
    named = title_key.startswith(name_key) and len(re.sub(r"\d", "", name_key)) >= 3
    if make_key and make_key not in title_key and not named:
        return False
    if len(name_key) >= 2 and name_key in title_key:
        return True
    if kind == "film":
        base = name_key.removesuffix(str(speed or ""))
        base = re.sub(r"\d+$", "", base)
        return len(base) >= 3 and base in title_key
    return False


def _parsed_page(fetch: Fetch, api: str, title: str) -> Optional[Tuple[str, str]]:
    data = fetch(
        api,
        {
            "action": "parse",
            "page": title,
            "prop": "text",
            "redirects": 1,
            "format": "json",
            "formatversion": 2,
        },
    )
    parsed = data.get("parse")
    if not parsed:
        return None
    text = html_text(parsed.get("text") or "")
    return (parsed["title"], text) if text else None


def _camera_wiki_page(fetch: Fetch, title: str) -> Optional[Source]:
    page = _parsed_page(fetch, CAMERA_WIKI, title)
    if not page:
        return None
    url = CAMERA_WIKI_PAGE + page[0].replace(" ", "_")
    return Source("camera-wiki", page[0], url, page[1])


def _wikipedia_page(fetch: Fetch, title: str) -> Optional[Source]:
    page = _parsed_page(fetch, WIKIPEDIA, title)
    if not page:
        return None
    url = WIKIPEDIA_PAGE + page[0].replace(" ", "_")
    return Source("wikipedia", page[0], url, page[1])


def focus_text(text: str, terms: List[str], limit: int = SOURCE_CHARS) -> str:
    """A long page cut to its start and the passages around each mention of the
    terms, so a list of films keeps the rows about this one."""
    if len(text) <= limit:
        return text
    spans = []
    for term in terms:
        words = [re.escape(w) for w in re.split(r"[\s\-]+", term) if w]
        if words:
            pattern = r"[\s\-]*".join(words)
            spans += [m.span() for m in re.finditer(pattern, text, re.IGNORECASE)]
    if not spans:
        return text[:limit]
    windows: List[List[int]] = []
    for start, end in sorted(spans):
        start, end = max(0, start - 400), min(len(text), end + 800)
        if windows and start <= windows[-1][1]:
            windows[-1][1] = max(windows[-1][1], end)
        else:
            windows.append([start, end])
    out = text[:800]
    for start, end in windows:
        start = max(start, 800)
        if start < end:
            out += "\n...\n" + text[start:end]
        if len(out) >= limit:
            break
    return out[:limit]


def _search(fetch: Fetch, api: str, query: str) -> List[str]:
    data = fetch(
        api,
        {
            "action": "query",
            "list": "search",
            "srsearch": query,
            "srlimit": 5,
            "format": "json",
        },
    )
    return [hit["title"] for hit in (data.get("query") or {}).get("search") or []]


def find_source(item: Entry, fetch: Fetch) -> Optional[Source]:
    """The camera-wiki page for the entry, else its Wikipedia page. A triage item can
    pin a page with "source": "Title" (camera-wiki) or "wikipedia:Title", and say
    what to look for on a long page with "focus": ["Acros II"]."""
    kind = item["kind"]
    name = item[name_field(kind)]
    source = _find_page(item, fetch, kind, name)
    if source:
        source.text = focus_text(source.text, item.get("focus") or [name])
    return source


def _find_page(item: Entry, fetch: Fetch, kind: str, name: str) -> Optional[Source]:
    pinned = item.get("source")
    if pinned:
        if pinned.startswith("wikipedia:"):
            return _wikipedia_page(fetch, pinned.removeprefix("wikipedia:"))
        return _camera_wiki_page(fetch, pinned)
    # The wikis often name the make differently ("Fujica GS645S"), so a distinctive
    # model is also searched on its own
    queries = [f"{item['make']} {name}"]
    if len(normalize_key(name)) >= 4 and re.search(r"\d", name):
        queries.append(name)
    for api, page in ((CAMERA_WIKI, _camera_wiki_page), (WIKIPEDIA, _wikipedia_page)):
        tried = set()
        for query in queries:
            for title in _search(fetch, api, query):
                fits = title_fits(title, kind, name, item.get("speed"), item["make"])
                if title in tried or not fits:
                    continue
                tried.add(title)
                source = page(fetch, title)
                if source:
                    return source
    return None


def pick_examples(
    item: Entry, entries: List[Entry], count: int = EXAMPLES
) -> List[Entry]:
    """Existing entries to copy the style from: the same make first, then entries
    spread through the catalog so the picks are the same on every run."""
    same = [e for e in entries if e["make"] == item["make"]]
    picks = same[:count]
    step = max(1, len(entries) // (count + 1))
    for e in entries[step::step]:
        if len(picks) >= count:
            break
        if e not in picks:
            picks.append(e)
    return picks


def draft_prompt(item: Entry, source: Source, examples: List[Entry]) -> Tuple[str, str]:
    kind = item["kind"]
    words = TARGET_WORDS[kind]
    name = f"{item['make']} {item[name_field(kind)]}"
    fields = '{"description": "..."}'
    if kind == "film":
        fields = '{"description": "...", "color_type": "color" or "bw"}'
    system = (
        f"You write the short {kind} descriptions for an analog photography catalog. "
        f"Write one plain-text paragraph of at most {words} words in the same style and "
        "voice as the examples: what it is, when and by whom it was made, its main "
        "features, what it is known for. Write about this exact model, not its family "
        "or brand in general. Use only facts stated in the source. If the source "
        "doesn't say something, leave it out rather than guess, and write less when "
        "the source has less to say: never pad. No marketing language, no lists, no "
        f"markdown. Reply with JSON only: {fields}"
    )
    lines = [f"Entry: {name}"]
    if kind == "film":
        lines.append(f"Box speed: ISO {item.get('speed')}")
    lines.append("")
    lines.append("Examples of existing descriptions:")
    for e in examples:
        lines.append(f"- {e['make']} {e[name_field(kind)]}: {e['description']}")
    lines.append("")
    lines.append(f"Source ({source.site}, {source.title}):")
    lines.append(source.text)
    return system, "\n".join(lines)


def parse_draft(text: str, kind: str) -> Dict[str, str]:
    """The LLM's reply as {"description", "color_type"}, or ValueError."""
    text = re.sub(r"^```(?:json)?\s*|\s*```$", "", text.strip())
    try:
        data = json.loads(text)
    except json.JSONDecodeError as e:
        raise ValueError(f"reply is not JSON: {e}") from e
    if not isinstance(data, dict):
        raise ValueError("reply is not a JSON object")
    description = " ".join(str(data.get("description") or "").split())
    low, high = WORD_RANGE[kind]
    words = len(description.split())
    if not low <= words <= high:
        raise ValueError(f"description has {words} words, want {low} to {high}")
    out = {"description": description}
    if kind == "film":
        color = str(data.get("color_type") or "").lower()
        if color not in COLOR_TYPES:
            raise ValueError(f"color_type {color!r}, want color or bw")
        out["color_type"] = color
    return out


def _new_entry(item: Entry) -> Entry:
    aliases = list(item.get("aliases") or [])
    if item["kind"] == "camera":
        entry = {"make": item["make"], "model": item["model"]}
    else:
        entry = {
            "type": item["type"],
            "make": item["make"],
            "speed": item["speed"],
            "color_type": item["color_type"],
        }
    if aliases:
        entry["aliases"] = aliases
    entry["description"] = item["description"]
    return entry


def _insert(entries: List[Entry], entry: Entry) -> None:
    last = max(
        (i for i, e in enumerate(entries) if e["make"] == entry["make"]), default=None
    )
    if last is None:
        entries.append(entry)
    else:
        entries.insert(last + 1, entry)


def apply_drafts(
    cameras: List[Entry], films: List[Entry], items: List[Entry]
) -> Tuple[List[Entry], List[Entry]]:
    """The catalog with the drafted entries and aliases added. New entries go after
    the last entry of their make. Existing entries only ever gain aliases."""
    catalog = {"camera": copy.deepcopy(cameras), "film": copy.deepcopy(films)}

    def find(kind: str, make: str, name: str) -> Optional[Entry]:
        key = normalize_key(f"{make} {name}")
        for e in catalog[kind]:
            if normalize_key(f"{e['make']} {e[name_field(kind)]}") == key:
                return e
        return None

    for item in items:
        if item.get("action") != "new":
            continue
        kind = item["kind"]
        if not item.get("description"):
            raise ValueError(
                f"{kind} {item['make']} {item[name_field(kind)]}: no description"
            )
        if find(kind, item["make"], item[name_field(kind)]):
            raise ValueError(
                f"{kind} {item['make']} {item[name_field(kind)]}: already in the catalog"
            )
        _insert(catalog[kind], _new_entry(item))

    for item in items:
        if item.get("action") != "alias":
            continue
        kind = item["kind"]
        target = find(kind, item["make"], item["target"])
        if target is None:
            raise ValueError(
                f"{kind} {item['make']} {item['target']}: alias target missing"
            )
        aliases = target.setdefault("aliases", [])
        for alias in item["aliases"]:
            if alias not in aliases:
                aliases.append(alias)
        if "description" in target:
            target["description"] = target.pop("description")

    return catalog["camera"], catalog["film"]


def review_markdown(items: List[Entry]) -> str:
    """One block per drafted entry, for reading and editing before apply."""
    lines = ["# Catalog drafts", ""]
    for item in items:
        if item.get("action") != "new":
            continue
        kind = item["kind"]
        name = f"{item['make']} {item[name_field(kind)]}"
        lines.append(f"## {kind}: {name}")
        if kind == "film":
            lines.append(f"ISO {item.get('speed')}, {item.get('color_type') or '?'}")
        if item.get("source_url"):
            lines.append(f"Source: [{item.get('source_title')}]({item['source_url']})")
        else:
            lines.append("Source: **no source**")
        if item.get("error"):
            lines.append(f"Error: {item['error']}")
        description = item.get("description") or ""
        lines.append(f"Words: {len(description.split())}")
        lines.append("")
        lines.append(description)
        lines.append("")
    return "\n".join(lines)
