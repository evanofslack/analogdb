"""Upkeep over the whole post archive: rematching metadata and pruning the
catalog after catalog changes, repairing posts missing a caption or an image
vector, and extracting post colors again.

These assets return only metadata, so nothing large goes through the IO
manager. Posts are processed in chunks so a failure late in a long run keeps
the work already written.
"""

import time
from dataclasses import dataclass
from typing import Dict, Iterator, List, Optional, Sequence

import analogdb.models as adb
import dagster as dg
from analogdb.client import Client, Deleted
from scrape.tagging import TAGGER_VERSION, TEXT_ONLY_SUFFIX, TagInput, medium_url

from .assets import error_detail, upload_cameras, upload_films
from .catalog import Retired, camera_key, catalog_for_matching, retired_entries
from .convert import convert_caption, convert_color, convert_keyword
from .metadata_flow import (
    WriteStats,
    catalog_matcher,
    extraction_record,
    load_catalog,
    patch_groups,
    plan_patch,
    rematch,
)
from .resources import (
    AnalogDBResource,
    CamerasJsonResource,
    FilmsJsonResource,
    ImageProcessorResource,
    TaggerResource,
)

BACKFILL_CHUNK = 200
MAX_POSTS = 1_000_000
ENCODE_LOG_EVERY = 1000


def chunks(items: Sequence, size: int) -> Iterator[Sequence]:
    for start in range(0, len(items), size):
        yield items[start : start + size]


@dg.asset(group_name="backfill", deps=[upload_cameras, upload_films])
def rematch_post_metadata(
    context: dg.AssetExecutionContext,
    analogdb: AnalogDBResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
) -> dg.MaterializeResult:
    """Match every stored extraction again with the current catalog and aliases,
    without LLM calls, and patch the posts whose values change. Run it after a
    catalog upload or a matcher change."""
    client = analogdb.client()
    matcher = catalog_matcher(load_catalog(context, client, cameras_json, films_json))
    posts = {p.id: p for p in client.get_posts_all(count=MAX_POSTS)}
    context.log.info(f"Fetched {len(posts)} posts")

    stats = WriteStats()
    changed: List[adb.PostExtraction] = []
    for extraction in client.iter_extractions(full=True):
        post = posts.get(extraction.post_id)
        if post is None:
            continue
        stats.posts += 1
        result = rematch(matcher, extraction)
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
        record = extraction_record(
            post.id, result, extraction.model, extraction.input or ""
        )
        record.extractor_version = extraction.extractor_version
        stats.records.append(record)
        if record.unmatched != (extraction.unmatched or []):
            changed.append(record)

    if changed:
        try:
            stats.stored, _ = client.upsert_extractions(changed)
        except Exception as e:
            stats.store_failed = len(changed)
            context.log.error(f"Failed to store {len(changed)} extractions, error={e}")

    return finish(context, "rematch metadata", stats)


def finish(
    context: dg.AssetExecutionContext, name: str, stats: WriteStats
) -> dg.MaterializeResult:
    metadata = stats.metadata()
    context.log.info(
        f"Finished {name}, totals: posts={stats.posts}, patched={stats.patched}, "
        f"unchanged={stats.unchanged}, llm_failed={stats.llm_failed}, "
        f"patch_failed={stats.patch_failed}, extractions_stored={stats.stored}"
    )
    if stats.failures():
        raise dg.Failure(
            description=f"{name}: {stats.patch_failed} patches and {stats.store_failed} extractions failed",
            metadata=metadata,
        )
    return dg.MaterializeResult(metadata=metadata)


class PruneCatalogConfig(dg.Config):
    dry_run: bool = True


@dataclass
class Move:
    post_id: int
    retired: Retired
    patch: adb.PostPatch


def straggler_moves(
    posts: Sequence[adb.Post], retired: Sequence[Retired]
) -> List[Move]:
    """Posts still on a retired name, each with the patch to its target. These are
    posts the rematch can't move: the extraction didn't mention a camera or film."""
    by_name = {(r.kind, r.make, r.name): r for r in retired}
    moves = []
    for p in posts:
        camera = by_name.get(
            ("camera", *camera_key(p.camera_make or "", p.camera_model or ""))
        )
        if camera:
            t = camera.target
            patch = adb.PostPatch(
                camera_make=t["make"].lower(), camera_model=t["model"].lower()
            )
            moves.append(Move(p.id, camera, patch))
        film = by_name.get(("film", *camera_key(p.film_make or "", p.film_type or "")))
        if film:
            t = film.target
            patch = adb.PostPatch(
                film_make=t["make"].lower(),
                film_type=t["type"].lower(),
                film_speed=t["speed"],
            )
            moves.append(Move(p.id, film, patch))
    return moves


def _retired_table(rows: Sequence[Retired], results: Dict[int, str]) -> str:
    if not rows:
        return "None."
    lines = ["| kind | name | alias of | result |", "|---|---|---|---|"]
    for r in rows:
        target = f"{r.target['make']} {r.target.get('model') or r.target.get('type')}"
        result = results.get(id(r), "dry run")
        lines.append(f"| {r.kind} | {r.make} {r.name} | {target} | {result} |")
    return "\n".join(lines)


@dg.asset(group_name="catalog", deps=[rematch_post_metadata])
def prune_catalog(
    context: dg.AssetExecutionContext,
    config: PruneCatalogConfig,
    analogdb: AnalogDBResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
) -> dg.MaterializeResult:
    """Retire renamed and merged catalog names: live entries the JSON lists as an
    alias of another. Moves posts still on them to the target, then deletes them.
    Live entries the JSON doesn't know are listed, never deleted. Dry run by default."""
    client = analogdb.client()
    camera_entries, film_entries = cameras_json.client(), films_json.client()
    live_cameras, live_films = client.get_cameras(), client.get_films()
    retired = retired_entries(camera_entries, film_entries, live_cameras, live_films)
    unknown = catalog_for_matching(
        camera_entries, film_entries, live_cameras, live_films
    ).not_in_json
    context.log.info(f"Found {len(retired)} retired and {len(unknown)} unknown entries")

    moves: List[Move] = []
    if retired:
        moves = straggler_moves(client.get_posts_all(count=MAX_POSTS), retired)

    results: Dict[int, str] = {}
    counts = {
        "deleted": 0,
        "in_use": 0,
        "missing": 0,
        "patch_failed": 0,
        "delete_failed": 0,
    }
    if not config.dry_run:
        for m in moves:
            try:
                client.patch_post(m.post_id, m.patch)
            except Exception as e:
                counts["patch_failed"] += 1
                context.log.error(
                    f"Failed to move post off retired name, id={m.post_id}, {error_detail(e)}"
                )
        for r in retired:
            delete = client.delete_camera if r.kind == "camera" else client.delete_film
            try:
                result = delete(r.id)
            except Exception as e:
                counts["delete_failed"] += 1
                results[id(r)] = "failed"
                context.log.error(
                    f"Failed to delete {r.kind} {r.make} {r.name}, {error_detail(e)}"
                )
                continue
            results[id(r)] = result.value
            key = {Deleted.DELETED: "deleted", Deleted.IN_USE: "in_use"}.get(
                result, "missing"
            )
            counts[key] += 1

    metadata = {
        "dry_run": config.dry_run,
        "retired": len(retired),
        "posts_moved_camera": sum(1 for m in moves if m.retired.kind == "camera"),
        "posts_moved_film": sum(1 for m in moves if m.retired.kind == "film"),
        **counts,
        "retired_entries": dg.MetadataValue.md(_retired_table(retired, results)),
        "unknown_entries": dg.MetadataValue.md(
            "\n".join(f"- {name}" for name in unknown) or "None."
        ),
    }
    context.log.info(f"Finished prune catalog: {metadata}")
    if counts["patch_failed"] or counts["delete_failed"]:
        raise dg.Failure(
            description=f"prune catalog: {counts['patch_failed']} patches and "
            f"{counts['delete_failed']} deletes failed",
            metadata=metadata,
        )
    return dg.MaterializeResult(metadata=metadata)


class TagMissingCaptionsConfig(dg.Config):
    limit: Optional[int] = None
    retry_text_only: bool = False
    fail_on_error: bool = True


@dataclass
class CaptionStats:
    posts: int = 0
    written: int = 0
    text_only: int = 0
    tag_failed: int = 0
    patch_failed: int = 0
    not_found: int = 0

    def metadata(self) -> Dict[str, int]:
        return dict(self.__dict__)


def missing_caption_ids(
    client: Client, version: str, retry_text_only: bool
) -> List[int]:
    """Posts without a caption of this version. Posts that only got tags from
    their title are left out unless retry_text_only is set."""
    ids = client.get_missing_captions(version)
    if retry_text_only:
        return ids
    has_text_only = set(ids) - set(
        client.get_missing_captions(version + TEXT_ONLY_SUFFIX)
    )
    return [id for id in ids if id not in has_text_only]


@dg.asset(group_name="backfill")
def tag_missing_post_captions(
    context: dg.AssetExecutionContext,
    config: TagMissingCaptionsConfig,
    analogdb: AnalogDBResource,
    tagger: TaggerResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
) -> dg.MaterializeResult:
    """Caption and tag every post missing a caption of the current tagger
    version, and replace its keywords with the tags. Written posts leave the
    missing list, so a rerun picks up where the last one stopped. Patch failures
    fail the run unless fail_on_error is off."""
    client = analogdb.client()
    ids = missing_caption_ids(client, TAGGER_VERSION, config.retry_text_only)
    if config.limit is not None:
        ids = ids[: config.limit]
    context.log.info(f"Found {len(ids)} posts missing a {TAGGER_VERSION} caption")
    stats = CaptionStats()
    if not ids:
        return dg.MaterializeResult(metadata=stats.metadata())

    posts = {p.id: p for p in client.get_posts_all(count=MAX_POSTS)}
    tagger_client = tagger.client(cameras_json.client(), films_json.client())

    for chunk in chunks(ids, BACKFILL_CHUNK):
        found = [posts[id] for id in chunk if id in posts]
        stats.not_found += len(chunk) - len(found)
        stats.posts += len(found)
        inputs = []
        tagged = []
        for p in found:
            try:
                url = medium_url(p.images or [])
            except ValueError:
                stats.tag_failed += 1
                context.log.warning(f"Failed to tag post, id={p.id}, no medium image")
                continue
            inputs.append(
                TagInput(
                    image_url=url,
                    title=p.title or "",
                    description=p.description,
                    grayscale=bool(p.grayscale),
                )
            )
            tagged.append(p)

        results = tagger_client.tag_all(inputs, tagger.concurrency)
        for p, r in zip(tagged, results, strict=True):
            if isinstance(r, Exception):
                stats.tag_failed += 1
                context.log.warning(f"Failed to tag post, id={p.id}, error={r}")
                continue
            patch = adb.PostPatch(
                caption=convert_caption(r.post_caption()),
                keywords=[convert_keyword(k) for k in r.keywords()],
            )
            try:
                client.patch_post(p.id, patch)
            except Exception as e:
                stats.patch_failed += 1
                context.log.error(
                    f"Failed to patch post caption, id={p.id}, {error_detail(e)}"
                )
                continue
            stats.written += 1
            if r.text_only:
                stats.text_only += 1
        context.log.info(
            f"Tag missing captions progress: {stats.posts} of {len(ids)} posts, "
            f"written={stats.written}, text_only={stats.text_only}, "
            f"failed={stats.tag_failed + stats.patch_failed}"
        )

    metadata = stats.metadata()
    context.log.info(f"Finished tag missing captions: {metadata}")
    if stats.patch_failed and config.fail_on_error:
        raise dg.Failure(
            description=f"tag missing captions: {stats.patch_failed} patches failed",
            metadata=metadata,
        )
    return dg.MaterializeResult(metadata=metadata)


class ReencodeVectorsConfig(dg.Config):
    batch_size: int = 20
    ids: Optional[List[int]] = None
    limit: Optional[int] = None
    fail_on_error: bool = True


def reencode_ids(config: ReencodeVectorsConfig, client: Client) -> List[int]:
    if config.ids is not None:
        ids = config.ids
    else:
        ids = client.get_missing_vectors()
    if config.limit is not None:
        ids = ids[: config.limit]
    return ids


def encode_batch(
    context: dg.AssetExecutionContext, client: Client, ids: Sequence[int], size: int
) -> List[int]:
    try:
        return client.encode_posts(list(ids), size)
    except Exception as e:
        context.log.warning(
            f"Failed to encode batch of {len(ids)} from id={ids[0]}, {error_detail(e)}"
        )
        return list(ids)


@dg.asset(group_name="backfill")
def reencode_post_vectors(
    context: dg.AssetExecutionContext,
    config: ReencodeVectorsConfig,
    analogdb: AnalogDBResource,
) -> dg.MaterializeResult:
    """Encode the image vectors of posts missing one, or of the configured ids.
    Failed ids are retried once on their own. Remaining failures fail the run
    unless fail_on_error is off."""
    client = analogdb.client()
    ids = reencode_ids(config, client)
    if not ids:
        context.log.info("No posts missing a vector")
        return dg.MaterializeResult(metadata={"posts": 0})
    context.log.info(f"Encoding {len(ids)} posts in batches of {config.batch_size}")

    failed: List[int] = []
    done = 0
    start = time.monotonic()
    for batch in chunks(ids, config.batch_size):
        failed += encode_batch(context, client, batch, config.batch_size)
        before = done
        done += len(batch)
        if done // ENCODE_LOG_EVERY > before // ENCODE_LOG_EVERY or done == len(ids):
            rate = done / max(time.monotonic() - start, 1e-9)
            context.log.info(
                f"Encoded {done} of {len(ids)} posts, {len(failed)} failed, "
                f"{rate:.1f} posts/s"
            )

    retried = len(failed)
    remaining = [id for id in failed if encode_batch(context, client, [id], 1)]
    metadata = {
        "posts": len(ids),
        "retried": retried,
        "failed": len(remaining),
    }
    context.log.info(f"Finished reencode vectors: {metadata}")
    if remaining:
        context.log.error(f"Failed to encode posts: {remaining}")
        failed_metadata = {**metadata, "failed_ids": str(remaining)}
        if config.fail_on_error:
            raise dg.Failure(
                description=f"reencode vectors: {len(remaining)} of {len(ids)} posts failed",
                metadata=failed_metadata,
            )
        return dg.MaterializeResult(metadata=failed_metadata)
    return dg.MaterializeResult(metadata=metadata)


class ReextractColorsConfig(dg.Config):
    ids: Optional[List[int]] = None
    limit: Optional[int] = None


@dataclass
class ColorStats:
    posts: int = 0
    patched: int = 0
    no_image: int = 0
    extract_failed: int = 0
    patch_failed: int = 0
    not_found: int = 0

    def metadata(self) -> Dict[str, int]:
        return dict(self.__dict__)


def low_url(images: Sequence[adb.Image]) -> Optional[str]:
    for image in images:
        if image.resolution == "low" and image.url:
            return image.url
    return None


@dg.asset(group_name="backfill")
def reextract_post_colors(
    context: dg.AssetExecutionContext,
    config: ReextractColorsConfig,
    analogdb: AnalogDBResource,
    image_processor: ImageProcessorResource,
) -> dg.MaterializeResult:
    """Extract the palette of every post, or of the configured ids, from its low
    res image and patch the post colors. Failures are counted and listed, and
    never fail the run, so a rerun with the failed ids picks them up."""
    client = analogdb.client()
    processor = image_processor.client()
    posts = {p.id: p for p in client.get_posts_all(count=MAX_POSTS)}
    ids = config.ids if config.ids is not None else sorted(posts)
    if config.limit is not None:
        ids = ids[: config.limit]
    context.log.info(f"Extracting colors of {len(ids)} posts")

    stats = ColorStats()
    failed: List[int] = []
    start = time.monotonic()
    for chunk in chunks(ids, BACKFILL_CHUNK):
        for id in chunk:
            post = posts.get(id)
            if post is None:
                stats.not_found += 1
                continue
            stats.posts += 1
            url = low_url(post.images or [])
            if url is None:
                stats.no_image += 1
                failed.append(id)
                context.log.warning(f"Failed to extract colors, id={id}, no low image")
                continue
            try:
                colors = processor.download_colors(url)
            except Exception as e:
                stats.extract_failed += 1
                failed.append(id)
                context.log.warning(f"Failed to extract colors, id={id}, error={e}")
                continue
            patch = adb.PostPatch(colors=[convert_color(c) for c in colors])
            try:
                client.patch_post(id, patch)
            except Exception as e:
                stats.patch_failed += 1
                failed.append(id)
                context.log.error(
                    f"Failed to patch post colors, id={id}, {error_detail(e)}"
                )
                continue
            stats.patched += 1
        rate = stats.posts / max(time.monotonic() - start, 1e-9)
        context.log.info(
            f"Reextract colors progress: {stats.posts} of {len(ids)} posts, "
            f"patched={stats.patched}, failed={len(failed)}, {rate:.1f} posts/s"
        )

    metadata: Dict[str, object] = {**stats.metadata()}
    if failed:
        metadata["failed_ids"] = str(failed)
        context.log.error(f"Failed to reextract colors of posts: {failed}")
    context.log.info(f"Finished reextract colors: {stats.metadata()}")
    return dg.MaterializeResult(metadata=metadata)
