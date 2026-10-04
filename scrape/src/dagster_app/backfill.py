"""Backfills over the post archive.

Backfill assets use the daily partitions with a single run policy: any range of
days (a day, a month, years) launched from the Dagster UI runs as one run over
that time window. They return only metadata, so nothing large goes through the
IO manager. Posts are processed in chunks so a failure late in a long range
keeps the work already written.

A new backfill is one asset that calls `range_posts`, optionally
`range_comments`, and does its work per chunk.
"""

import json
import time
from dataclasses import dataclass
from typing import Dict, Iterator, List, Optional, Sequence

import analogdb.models as adb
import dagster as dg
from analogdb.client import Client
from scrape.constants import AWS_BUCKET_COMMENTS
from scrape.models import RedditComment
from scrape.s3 import upload_comments
from scrape.tagging import TAGGER_VERSION, TEXT_ONLY_SUFFIX, TagInput, medium_url

from .assets import daily_partitions, error_detail
from .convert import convert_caption, convert_keyword
from .metadata_flow import (
    WriteStats,
    catalog_matcher,
    extract_and_write,
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
    MetadataResource,
    RedditResource,
    StorageResource,
    TaggerResource,
)

BACKFILL_CHUNK = 200
MAX_POSTS = 1_000_000
MIN_POSTS_FOR_REDDIT_CHECK = 5
ENCODE_LOG_EVERY = 1000


def chunks(items: Sequence, size: int) -> Iterator[Sequence]:
    for start in range(0, len(items), size):
        yield items[start : start + size]


def range_window(context: dg.AssetExecutionContext) -> dg.TimeWindow:
    """The time window covering every partition in the run."""
    keys = context.partition_key_range
    first = daily_partitions.time_window_for_partition_key(keys.start)
    last = daily_partitions.time_window_for_partition_key(keys.end)
    return dg.TimeWindow(first.start, last.end)


def range_posts(context: dg.AssetExecutionContext, client: Client) -> List[adb.Post]:
    """Every post in the run's time window, oldest first."""
    window = range_window(context)
    start, end = int(window.start.timestamp()), int(window.end.timestamp())
    filter = adb.create_posts_filter(time_start=start, time_end=end)
    posts = client.get_posts_all(count=MAX_POSTS, filter=filter)
    posts.sort(key=lambda p: (p.timestamp, p.id))
    context.log.info(f"Fetched {len(posts)} posts from {window.start} to {window.end}")
    return posts


def range_comments(
    context: dg.AssetExecutionContext,
    posts: Sequence[adb.Post],
    storage: StorageResource,
    reddit: RedditResource,
) -> Dict[int, List[RedditComment]]:
    """Reddit comments per post, from the S3 comments cache, else from Reddit
    and written back to the cache. Posts Reddit fails on get no comments."""
    comments: Dict[int, List[RedditComment]] = {}
    cached = fetched = failed = 0
    scraper = None
    for p in posts:
        body = storage.get_object(AWS_BUCKET_COMMENTS, f"{p.id}.json")
        if body is not None:
            comments[p.id] = [RedditComment(**c) for c in json.loads(body)]
            cached += 1
            continue
        scraper = scraper or reddit.client()
        try:
            found = scraper.scrape_comments(p.permalink)
        except Exception as e:
            failed += 1
            comments[p.id] = []
            context.log.warning(f"Failed to fetch comments, id={p.id}, error={e}")
            continue
        comments[p.id] = found
        fetched += 1
        upload_comments(storage, p.id, found)

    context.log.info(f"Comments: {cached} cached, {fetched} fetched, {failed} failed")
    needed = fetched + failed
    if needed >= MIN_POSTS_FOR_REDDIT_CHECK and fetched == 0:
        raise dg.Failure(
            f"Reddit failed for all {needed} posts missing from the comments cache"
        )
    return comments


@dg.asset(
    partitions_def=daily_partitions,
    backfill_policy=dg.BackfillPolicy.single_run(),
    group_name="backfill",
)
def backfill_post_metadata(
    context: dg.AssetExecutionContext,
    analogdb: AnalogDBResource,
    metadata: MetadataResource,
    reddit: RedditResource,
    storage: StorageResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
) -> dg.MaterializeResult:
    """Extract camera, film and lens metadata for every post in the range with
    the title, description and OP comments, overwrite by the write rule, and
    store each extraction."""
    client = analogdb.client()
    posts = range_posts(context, client)
    catalog = load_catalog(context, client, cameras_json, films_json)
    extractor = metadata.client()

    stats = WriteStats()
    for chunk in chunks(posts, BACKFILL_CHUNK):
        comments = range_comments(context, chunk, storage, reddit)
        chunk_stats = extract_and_write(
            context, client, extractor, catalog, metadata.openai_model, chunk, comments
        )
        stats.add(chunk_stats)
        context.log.info(
            f"Metadata backfill progress: {stats.posts} of {len(posts)} posts"
        )

    return finish(context, "backfill metadata", stats)


@dg.asset(group_name="backfill")
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
        f"Finished {name}: posts={stats.posts}, patched={stats.patched}, "
        f"unchanged={stats.unchanged}, llm_failed={stats.llm_failed}, "
        f"patch_failed={stats.patch_failed}, extractions_stored={stats.stored}"
    )
    if stats.failures():
        raise dg.Failure(
            description=f"{name}: {stats.patch_failed} patches and {stats.store_failed} extractions failed",
            metadata=metadata,
        )
    return dg.MaterializeResult(metadata=metadata)


class BackfillCaptionsConfig(dg.Config):
    version: str = TAGGER_VERSION
    limit: Optional[int] = None
    retry_text_only: bool = False


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
def backfill_post_captions(
    context: dg.AssetExecutionContext,
    config: BackfillCaptionsConfig,
    analogdb: AnalogDBResource,
    tagger: TaggerResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
) -> dg.MaterializeResult:
    """Caption and tag every post missing a caption of the version, and replace
    its keywords with the tags. Written posts leave the missing list, so a rerun
    picks up where the last one stopped."""
    client = analogdb.client()
    ids = missing_caption_ids(client, config.version, config.retry_text_only)
    if config.limit is not None:
        ids = ids[: config.limit]
    context.log.info(f"Found {len(ids)} posts missing a {config.version} caption")
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
            f"Caption backfill progress: {stats.posts} of {len(ids)} posts, "
            f"written={stats.written}, text_only={stats.text_only}, "
            f"failed={stats.tag_failed + stats.patch_failed}"
        )

    metadata = stats.metadata()
    context.log.info(f"Finished backfill captions: {metadata}")
    if stats.patch_failed:
        raise dg.Failure(
            description=f"backfill captions: {stats.patch_failed} patches failed",
            metadata=metadata,
        )
    return dg.MaterializeResult(metadata=metadata)


class ReencodeVectorsConfig(dg.Config):
    batch_size: int = 20
    ids: Optional[List[int]] = None
    limit: Optional[int] = None
    missing_only: bool = True
    fail_on_error: bool = True


def reencode_ids(config: ReencodeVectorsConfig, client: Client) -> List[int]:
    if config.ids is not None:
        ids = config.ids
    elif config.missing_only:
        ids = client.get_missing_vectors()
    else:
        ids = client.get_post_ids()
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
    """Encode the image vectors of posts missing one, or of every post when
    missing_only is off, or of the configured ids. Failed ids are retried once
    on their own. Remaining failures fail the run unless fail_on_error is off."""
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
