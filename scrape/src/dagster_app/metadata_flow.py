import hashlib
from collections import Counter, defaultdict
from dataclasses import dataclass, field
from typing import Dict, List, Optional, Sequence

import analogdb.models as adb
import dagster as dg
from analogdb.client import Client
from scrape.catalog_match import CatalogMatcher
from scrape.metadata import EXTRACTOR_VERSION, MetadataExtractor, post_text
from scrape.models import MatchResult, MetadataPost, RedditComment

from .catalog import MatchingCatalog, catalog_for_matching
from .resources import CamerasJsonResource, FilmsJsonResource

COMMENT_CHARS = 1500
REMOVED_COMMENTS = {"[deleted]", "[removed]"}

CAMERA_FIELDS = ("camera_make", "camera_model")
FILM_FIELDS = ("film_make", "film_type", "film_speed")
LENS_FIELDS = ("focal_length", "aperture")


def op_comments(comments: Sequence[RedditComment], author: str) -> List[str]:
    """The post author's own comments, oldest first, capped in total length."""
    author = author.removeprefix("u/")
    own = sorted(
        (c for c in comments if c.author.removeprefix("u/") == author),
        key=lambda c: c.time,
    )
    bodies: List[str] = []
    total = 0
    for c in own:
        body = " ".join(c.body.split())
        if not body or body in REMOVED_COMMENTS:
            continue
        body = body[: COMMENT_CHARS - total]
        bodies.append(body)
        total += len(body)
        if total >= COMMENT_CHARS:
            break
    return bodies


def metadata_post(
    post: adb.Post, comments: Sequence[RedditComment] = ()
) -> MetadataPost:
    return MetadataPost(
        title=post.title or "",
        description=post.description,
        op_comments=op_comments(comments, post.author or ""),
    )


def extraction_record(
    post_id: int, result: MatchResult, model: str, text: str
) -> adb.PostExtraction:
    return adb.PostExtraction(
        post_id=post_id,
        extractor_version=EXTRACTOR_VERSION,
        model=model,
        input=text,
        input_hash=hashlib.sha256(text.encode("utf-8")).hexdigest(),
        raw=result.raw or {},
        unmatched=[u.to_dict() for u in result.unmatched],
    )


def _norm(value):
    if value is None or value == "":
        return None
    return value.strip().lower() if isinstance(value, str) else value


def plan_patch(current: adb.Post, result: MatchResult) -> Optional[adb.PostPatch]:
    """The write rule. When the post mentions a camera (or film), the extractor
    decides the whole group: set what it found, clear what it didn't, since an
    old value there is likely a wrong guess. With no mention, old values stay.
    Lens values are set when found and otherwise kept."""
    raw = result.raw or {}
    proposed = result.proposed
    changes: Dict[str, object] = {}
    clear: List[str] = []

    def decide(fields, may_clear: bool):
        for f in fields:
            new, old = getattr(proposed, f), getattr(current, f, None)
            if _norm(new) == _norm(old):
                continue
            if new is not None:
                changes[f] = new
            elif may_clear:
                clear.append(f)

    if raw.get("cameras"):
        decide(CAMERA_FIELDS, may_clear=True)
    if raw.get("films"):
        decide(FILM_FIELDS, may_clear=True)
    decide(LENS_FIELDS, may_clear=False)

    if not changes and not clear:
        return None
    return adb.PostPatch(**changes, clear=clear or None)


def patch_groups(patch: adb.PostPatch) -> List[str]:
    fields = set(patch.to_dict()) | set(patch.clear or [])
    groups = []
    for name, group in (
        ("camera", CAMERA_FIELDS),
        ("film", FILM_FIELDS),
        ("lens", LENS_FIELDS),
    ):
        if fields & set(group):
            groups.append(name)
    return groups


def unmatched_report(records: Sequence[adb.PostExtraction], top: int = 50) -> str:
    """Markdown table of the most common mentions missing from the catalog."""
    posts: Dict[tuple, set] = defaultdict(set)
    spellings: Dict[tuple, Counter] = defaultdict(Counter)
    for r in records:
        for u in r.unmatched or []:
            key = (u.get("kind"), u.get("key"))
            if not key[1]:
                continue
            posts[key].add(r.post_id)
            spellings[key][u.get("raw") or key[1]] += 1
    if not posts:
        return "No unmatched mentions."
    ranked = sorted(posts.items(), key=lambda kv: (-len(kv[1]), kv[0]))[:top]
    lines = ["| kind | spelling | posts | examples |", "|---|---|---|---|"]
    for (kind, key), ids in ranked:
        spelling = spellings[(kind, key)].most_common(1)[0][0]
        examples = ", ".join(str(i) for i in sorted(ids, reverse=True)[:3])
        lines.append(f"| {kind} | {spelling} | {len(ids)} | {examples} |")
    return "\n".join(lines)


@dataclass
class WriteStats:
    posts: int = 0
    patched: int = 0
    unchanged: int = 0
    llm_failed: int = 0
    patch_failed: int = 0
    stored: int = 0
    store_failed: int = 0
    groups: Counter = field(default_factory=Counter)
    records: List[adb.PostExtraction] = field(default_factory=list)

    def add(self, other: "WriteStats") -> None:
        for name in (
            "posts",
            "patched",
            "unchanged",
            "llm_failed",
            "patch_failed",
            "stored",
            "store_failed",
        ):
            setattr(self, name, getattr(self, name) + getattr(other, name))
        self.groups.update(other.groups)
        self.records.extend(other.records)

    def metadata(self) -> Dict[str, object]:
        return {
            "posts": self.posts,
            "patched": self.patched,
            "unchanged": self.unchanged,
            "camera_patches": self.groups["camera"],
            "film_patches": self.groups["film"],
            "lens_patches": self.groups["lens"],
            "llm_failed": self.llm_failed,
            "patch_failed": self.patch_failed,
            "extractions_stored": self.stored,
            "extractions_failed": self.store_failed,
            "unmatched": dg.MetadataValue.md(unmatched_report(self.records)),
        }

    def failures(self) -> int:
        return self.patch_failed + self.store_failed


def load_catalog(
    context: dg.AssetExecutionContext,
    client: Client,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
) -> MatchingCatalog:
    catalog = catalog_for_matching(
        cameras_json.client(),
        films_json.client(),
        client.get_cameras(),
        client.get_films(),
    )
    if catalog.not_live:
        context.log.warning(
            f"{len(catalog.not_live)} catalog json entries aren't live and won't be matched, "
            "run upload_cameras and upload_films"
        )
    return catalog


def write_results(
    context: dg.AssetExecutionContext,
    client: Client,
    posts: Sequence[adb.Post],
    results: Sequence[MatchResult],
    texts: Sequence[str],
    model: str,
) -> WriteStats:
    """Patch each post by the write rule and store its extraction. Posts the LLM
    failed on get neither, so a later run picks them up."""
    stats = WriteStats(posts=len(posts))
    for post, result, text in zip(posts, results, texts, strict=True):
        if "llm_failed" in result.flags:
            stats.llm_failed += 1
            continue
        patch = plan_patch(post, result)
        if patch is None:
            stats.unchanged += 1
        else:
            try:
                client.patch_post(post.id, patch)
                stats.patched += 1
                stats.groups.update(patch_groups(patch))
            except Exception as e:
                stats.patch_failed += 1
                context.log.error(
                    f"Failed to patch post metadata, id={post.id}, error={e}"
                )
        stats.records.append(extraction_record(post.id, result, model, text))

    if stats.records:
        try:
            written, skipped = client.upsert_extractions(stats.records)
            stats.stored = written
            if skipped:
                context.log.info(f"Skipped extractions for deleted posts {skipped}")
        except Exception as e:
            stats.store_failed = len(stats.records)
            context.log.error(
                f"Failed to store {len(stats.records)} extractions, error={e}"
            )
    return stats


def extract_and_write(
    context: dg.AssetExecutionContext,
    client: Client,
    extractor: MetadataExtractor,
    catalog: MatchingCatalog,
    model: str,
    posts: Sequence[adb.Post],
    comments: Dict[int, List[RedditComment]],
) -> WriteStats:
    inputs = [metadata_post(p, comments.get(p.id, [])) for p in posts]
    results = extractor.extract(inputs, catalog.cameras, catalog.films, catalog.aliases)
    texts = [post_text(i) for i in inputs]
    return write_results(context, client, posts, results, texts, model)


def catalog_matcher(catalog: MatchingCatalog) -> CatalogMatcher:
    return CatalogMatcher(catalog.cameras, catalog.films, catalog.aliases)


def rematch(matcher: CatalogMatcher, extraction: adb.PostExtraction) -> MatchResult:
    """Match a stored extraction again with the current catalog, no LLM call. The
    stored input is the text the LLM saw, which is all grounding needs."""
    return matcher.match(extraction.raw, MetadataPost(title=extraction.input or ""))
