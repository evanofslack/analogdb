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
from typing import Dict, Iterator, List, Sequence

import analogdb.models as adb
import dagster as dg
from analogdb.client import Client
from scrape.constants import AWS_BUCKET_COMMENTS
from scrape.models import RedditComment

from .assets import daily_partitions
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
)

BACKFILL_CHUNK = 200
MAX_POSTS = 1_000_000
MIN_POSTS_FOR_REDDIT_CHECK = 5


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
        body = json.dumps([c.__dict__ for c in found]).encode("UTF-8")
        storage.put_object(
            AWS_BUCKET_COMMENTS, f"{p.id}.json", body, "application/json"
        )

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
