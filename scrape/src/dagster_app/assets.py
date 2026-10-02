import json
import time
from dataclasses import asdict
from typing import List, Tuple

import analogdb.models as adb
import dagster as dg
from analogdb.client import Client, Uploaded
from analogdb_generated.exceptions import ApiException
from scrape.models import (
    MetadataPost,
    PhotoMetadata,
    PostImages,
    RedditComment,
    RedditPost,
    new_post_create,
)
from scrape.s3 import upload_comments
from scrape.tagging import ImageTags, TagInput, medium_url

from .catalog import UploadPlan, camera_upload_plan, film_upload_plan
from .constants import SUBREDDITS
from .convert import convert_create
from .metadata_flow import extract_and_write, load_catalog
from .resources import (
    AnalogDBResource,
    CamerasJsonResource,
    FilmsJsonResource,
    ImageProcessorResource,
    MetadataResource,
    RedditResource,
    StorageResource,
    TaggerResource,
)
from .result import Result, ResultDagsterType, Status

daily_partitions = dg.TimeWindowPartitionsDefinition(
    start="2018-01-01-00:00",
    cron_schedule="0 0 * * *",
    fmt="%Y-%m-%d",
)

MIN_POSTS_FOR_COMMENTS = 5


def add_result_metadata(context: dg.AssetExecutionContext, result: Result) -> None:
    context.add_output_metadata(
        {"success": result.successful_count(), "failed": result.failed_count()}
    )


def error_detail(e: Exception) -> str:
    if isinstance(e, ApiException):
        return f"status={e.status}, body={e.body}"
    return f"error={e}"


def patch_posts(
    context: dg.AssetExecutionContext,
    client: Client,
    patches: List[Tuple[int, adb.PostPatch]],
    name: str,
    delay: float = 0,
) -> None:
    failed = 0
    for id, p in patches:
        try:
            client.patch_post(id, p)
        except Exception as e:
            failed += 1
            context.log.error(
                f"Failed to patch post {name}, id={id}, {error_detail(e)}"
            )
        if delay:
            time.sleep(delay)

    context.log.info(
        f"Patched {len(patches) - failed} post {name} for partition {context.partition_key}"
    )
    if failed > 0:
        raise dg.Failure(
            description=f"Failed to patch {failed} of {len(patches)} post {name} for partition {context.partition_key}",
            metadata={"patched": len(patches) - failed, "failed": failed},
        )


@dg.asset(partitions_def=daily_partitions, group_name="analogdb")
def analogdb_posts(
    context: dg.AssetExecutionContext, analogdb: AnalogDBResource
) -> List[adb.Post]:
    window = context.partition_time_window
    start = int(window.start.timestamp())
    end = int(window.end.timestamp())

    filter = adb.create_posts_filter(time_start=start, time_end=end)
    posts = analogdb.client().get_posts_all(
        count=analogdb.batch_posts_count, filter=filter
    )
    context.log.info(
        f"Fetched {len(posts)} posts for partition {context.partition_key} ({window.start} to {window.end})"
    )
    return posts


@dg.asset(group_name="analogdb")
def analogdb_permalinks(
    context: dg.AssetExecutionContext, analogdb: AnalogDBResource
) -> List[str]:
    count = analogdb.permalink_posts_count
    links = analogdb.client().get_latest_links(count=count)
    context.log.info(f"Fetched {len(links)} post permalinks")
    return links


@dg.asset(
    dagster_type=ResultDagsterType,
    group_name="scrape",
    retry_policy=dg.RetryPolicy(
        max_retries=2, delay=60, backoff=dg.Backoff.EXPONENTIAL
    ),
)
def reddit_posts(
    context: dg.AssetExecutionContext,
    reddit: RedditResource,
    analogdb_permalinks: List[str],
) -> Result[RedditPost]:
    r = reddit.client()
    posts = []
    scrape_errors = []
    for subreddit, count in SUBREDDITS:
        scraped = r.scrape_posts(subreddit, count, analogdb_permalinks, "top")
        context.log.info(f"Scraped {len(scraped.posts)} posts from r/{subreddit}")
        posts += scraped.posts
        scrape_errors += scraped.errors

    data = {}
    status = {}
    errors = {}
    for p in posts:
        id = p.permalink
        data[id] = p
        status[id] = Status.SUCCESS
    for err in scrape_errors:
        context.log.warn(f"Scrape reddit post, {err}")
        status[err.permalink] = Status.FAILED
        errors[err.permalink] = err.msg

    result = Result(data=data, status=status, errors=errors)
    add_result_metadata(context, result)
    context.log.info(f"Scraped {result.successful_count()} successful posts")
    return result


@dg.asset(dagster_type=ResultDagsterType, group_name="scrape")
def title_metadatas(
    context: dg.AssetExecutionContext,
    metadata: MetadataResource,
    analogdb: AnalogDBResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
    reddit_posts,
) -> Result[PhotoMetadata]:
    posts = [p for _, p in reddit_posts.successful().items()]
    catalog = load_catalog(context, analogdb.client(), cameras_json, films_json)
    inputs = [MetadataPost(title=p.title, description=p.selftext) for p in posts]
    results = metadata.client().extract(
        inputs, catalog.cameras, catalog.films, catalog.aliases
    )

    failed = 0
    data = {}
    status = {}
    for p, r in zip(posts, results, strict=True):
        if "llm_failed" in r.flags:
            failed += 1
        context.log.debug(f"Extracted metadata from {p.title}, metadata: {r.proposed}")
        data[p.permalink] = r.proposed
        status[p.permalink] = Status.SUCCESS

    if failed > 0:
        context.log.warn(
            f"Failed to extract metadata for {failed} of {len(posts)} posts"
        )
    if posts and failed == len(posts):
        context.log.warn(
            "Failed to extract metadata for all posts, uploading without it"
        )

    with_metadata = len([r for r in results if not r.proposed.is_empty()])
    result = Result(data=data, status=status)
    context.add_output_metadata(
        {
            "success": result.successful_count(),
            "failed": result.failed_count(),
            "llm_failed": failed,
            "with_metadata": with_metadata,
        }
    )
    context.log.info(f"Extracted metadata from {result.successful_count()} posts")
    return result


@dg.asset(
    dagster_type=ResultDagsterType,
    group_name="scrape",
    retry_policy=dg.RetryPolicy(
        max_retries=2, delay=60, backoff=dg.Backoff.EXPONENTIAL
    ),
)
def post_images(
    context: dg.AssetExecutionContext,
    image_processor: ImageProcessorResource,
    storage: StorageResource,
    reddit_posts,
) -> Result[PostImages]:
    data = {}
    status = {}
    errors = {}
    processor = image_processor.client()
    for id, p in reddit_posts.successful().items():
        try:
            data[id] = processor.process(p, storage)
            status[id] = Status.SUCCESS
        except Exception as e:
            context.log.error(f"Failed to process images for {id}: {e}")
            status[id] = Status.FAILED
            errors[id] = str(e)

    result = Result(data=data, status=status, errors=errors)
    add_result_metadata(context, result)
    context.log.info(f"Processed images for {result.successful_count()} posts")
    return result


@dg.asset(dagster_type=ResultDagsterType, group_name="scrape")
def post_tags(
    context: dg.AssetExecutionContext,
    tagger: TaggerResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
    reddit_posts,
    post_images,
) -> Result[ImageTags]:
    ids = sorted(reddit_posts.successful_ids() & post_images.successful_ids())
    tagger_client = tagger.client(cameras_json.client(), films_json.client())

    data = {}
    status = {}
    errors = {}
    inputs = []
    tagged = []
    for id in ids:
        p = reddit_posts.data[id]
        try:
            url = medium_url(post_images.data[id].images)
        except ValueError as e:
            context.log.warn(f"Failed to tag {id}, uploading without tags: {e}")
            status[id] = Status.FAILED
            errors[id] = str(e)
            continue
        inputs.append(
            TagInput(
                image_url=url,
                title=p.title,
                description=p.selftext,
                grayscale=post_images.data[id].grayscale,
            )
        )
        tagged.append(id)

    results = tagger_client.tag_all(inputs, tagger.concurrency)
    text_only = 0
    for id, r in zip(tagged, results, strict=True):
        if isinstance(r, Exception):
            context.log.warn(f"Failed to tag {id}, uploading without tags: {r}")
            status[id] = Status.FAILED
            errors[id] = str(r)
            continue
        if r.text_only:
            text_only += 1
        data[id] = r
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status, errors=errors)
    context.add_output_metadata(
        {
            "success": result.successful_count(),
            "failed": result.failed_count(),
            "text_only": text_only,
        }
    )
    context.log.info(
        f"Tagged {result.successful_count()} posts, {text_only} from the title only"
    )
    return result


@dg.asset(group_name="scrape")
def final_posts(
    context: dg.AssetExecutionContext,
    reddit_posts,
    title_metadatas,
    post_images,
    post_tags,
):
    ids = (
        reddit_posts.successful_ids()
        & title_metadatas.successful_ids()
        & post_images.successful_ids()
    )
    context.log.info(f"Creating final posts for {len(ids)} posts")

    data = {}
    status = {}
    errors = {}
    tags = post_tags.successful()

    for id in ids:
        try:
            t = tags.get(id)
            if t is None:
                context.log.warn(f"Missing tags for {id}, uploading without them")
            final = new_post_create(
                post=reddit_posts.data[id],
                metadata=title_metadatas.data[id],
                images=post_images.data[id],
                keywords=t.keywords() if t else [],
                caption=t.post_caption() if t else None,
            )

            data[id] = final
            status[id] = Status.SUCCESS

        except Exception as e:
            context.log.error(f"Failed to create final post for {id}: {e}")
            status[id] = Status.FAILED
            errors[id] = str(e)

    result = Result(data=data, status=status, errors=errors)
    add_result_metadata(context, result)
    context.log.info(f"Created {result.successful_count()} final posts")
    return result


@dg.asset(
    group_name="scrape",
    retry_policy=dg.RetryPolicy(
        max_retries=2, delay=60, backoff=dg.Backoff.EXPONENTIAL
    ),
)
def upload_posts(
    context: dg.AssetExecutionContext, analogdb: AnalogDBResource, final_posts
) -> dg.MaterializeResult:
    adb = analogdb.client()
    created = 0
    exists = 0
    failed = 0
    for id, p in final_posts.successful().items():
        try:
            uploaded = adb.upload_post(convert_create(p))
        except Exception as e:
            failed += 1
            context.log.error(
                f"Failed to upload post, permalink={id}, {error_detail(e)}"
            )
            continue
        if uploaded == Uploaded.EXISTS:
            exists += 1
            context.log.info(f"Post already exists, permalink={id}")
        else:
            created += 1

    counts = {"created": created, "exists": exists, "failed": failed}
    context.log.info(
        f"Uploaded posts, created={created}, exists={exists}, failed={failed}"
    )
    if failed > 0:
        raise dg.Failure(
            description=f"Failed to upload {failed} of {created + exists + failed} posts",
            metadata=counts,
        )
    return dg.MaterializeResult(metadata=counts)


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def updated_post_scores(
    context: dg.AssetExecutionContext,
    analogdb_posts: List[adb.Post],
    reddit: RedditResource,
) -> List[Tuple[int, adb.PostPatch]]:
    r = reddit.client()
    patches: List[Tuple[int, adb.PostPatch]] = []
    for p in analogdb_posts:
        score = r.updated_score(p.permalink, p.score)
        if score is None:
            continue
        patches.append((p.id, adb.PostPatch(score=score)))
    context.log.info(f"Created {len(patches)} updated post scores")
    return patches


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_scores(
    context: dg.AssetExecutionContext,
    updated_post_scores: List[Tuple[int, adb.PostPatch]],
    analogdb: AnalogDBResource,
) -> None:
    if not updated_post_scores:
        context.log.info(
            f"No updated post scores to process for partition {context.partition_key}"
        )
        return

    patch_posts(context, analogdb.client(), updated_post_scores, "scores")


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def updated_post_descriptions(
    context: dg.AssetExecutionContext,
    analogdb_posts: List[adb.Post],
    reddit: RedditResource,
) -> List[Tuple[int, adb.PostPatch]]:
    r = reddit.client()
    patches: List[Tuple[int, adb.PostPatch]] = []
    for p in analogdb_posts:
        desc = r.updated_selftext(p.permalink)
        if desc is None or desc == "":
            continue
        patches.append((p.id, adb.PostPatch(description=desc)))
    context.log.info(f"Created {len(patches)} updated post descriptions")
    return patches


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_descriptions(
    context: dg.AssetExecutionContext,
    updated_post_descriptions: List[Tuple[int, adb.PostPatch]],
    analogdb: AnalogDBResource,
) -> None:
    if not updated_post_descriptions:
        context.log.info(
            f"No updated post descriptions to process for partition {context.partition_key}"
        )
        return

    patch_posts(context, analogdb.client(), updated_post_descriptions, "descriptions")


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def updated_reddit_comments(
    context: dg.AssetExecutionContext,
    analogdb_posts: List[adb.Post],
    reddit: RedditResource,
) -> List[Tuple[adb.Post, List[RedditComment]]]:
    post_comments: List[Tuple[adb.Post, List[RedditComment]]] = []
    if not analogdb_posts:
        context.log.info(
            f"No reddit comments to get for partition {context.partition_key}"
        )
        return post_comments

    r = reddit.client()
    for p in analogdb_posts:
        c = r.scrape_comments(p.permalink)
        post_comments.append((p, c))

    comment_count = sum(len(c) for _, c in post_comments)
    context.log.info(
        f"Scraped {comment_count} reddit comments from {len(post_comments)} posts for partition {context.partition_key}"
    )
    if len(post_comments) >= MIN_POSTS_FOR_COMMENTS and comment_count == 0:
        raise dg.Failure(
            f"No reddit comments found across {len(post_comments)} posts for partition {context.partition_key}"
        )
    context.add_output_metadata({"comment_count": comment_count})

    context.log.info(
        f"Created {len(post_comments)} updated reddit comments for partition {context.partition_key}"
    )
    return post_comments


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_metadata(
    context: dg.AssetExecutionContext,
    updated_reddit_comments: List[Tuple[adb.Post, List[RedditComment]]],
    analogdb: AnalogDBResource,
    metadata: MetadataResource,
    cameras_json: CamerasJsonResource,
    films_json: FilmsJsonResource,
) -> dg.MaterializeResult:
    """Second metadata pass two days after posting, now with the author's own
    comments. Overwrites by the write rule and stores the extraction."""
    if not updated_reddit_comments:
        context.log.info(f"No posts for metadata in partition {context.partition_key}")
        return dg.MaterializeResult(metadata={"posts": 0})

    client = analogdb.client()
    catalog = load_catalog(context, client, cameras_json, films_json)
    posts = [p for p, _ in updated_reddit_comments]
    comments = {p.id: c for p, c in updated_reddit_comments}
    stats = extract_and_write(
        context,
        client,
        metadata.client(),
        catalog,
        metadata.openai_model,
        posts,
        comments,
    )
    context.log.info(
        f"Metadata for partition {context.partition_key}: patched={stats.patched}, "
        f"unchanged={stats.unchanged}, llm_failed={stats.llm_failed}"
    )
    if stats.failures():
        raise dg.Failure(
            description=f"{stats.patch_failed} metadata patches and {stats.store_failed} extractions failed",
            metadata=stats.metadata(),
        )
    return dg.MaterializeResult(metadata=stats.metadata())


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def reddit_comments_to_s3(
    context: dg.AssetExecutionContext,
    updated_reddit_comments: List[Tuple[adb.Post, List[RedditComment]]],
    storage: StorageResource,
) -> None:
    if not updated_reddit_comments:
        context.log.info(
            f"No reddit comments to s3 for partition {context.partition_key}"
        )
        return

    for p, c in updated_reddit_comments:
        upload_comments(storage, p.id, c)

    context.log.info(
        f"Uploaded {len(updated_reddit_comments)} reddit comments to s3 for partition {context.partition_key}"
    )


@dg.asset(group_name="scrape")
def debug_posts(context: dg.AssetExecutionContext, final_posts) -> None:
    """Debug asset to inspect posts instead of uploading"""
    logger = context.log

    logger.info(f"Would upload {final_posts.successful_count()} posts")

    for i, (_, p) in enumerate(final_posts.successful().items()):
        logger.info(f"Post {i + 1}: {p.title} by {p.author} with score {p.score}")

    posts_dict = [asdict(p) for _, p in final_posts.successful().items()]
    with open("debug_posts.json", "w") as f:
        json.dump(posts_dict, f, indent=2)

    logger.info("Saved all posts to debug_posts.json")


@dg.asset(group_name="scrape")
def upload_films(
    context: dg.AssetExecutionContext,
    films_json: FilmsJsonResource,
    analogdb: AnalogDBResource,
) -> dg.MaterializeResult:
    analog = analogdb.client()
    plan = film_upload_plan(films_json.client(), analog.get_films())
    failed = 0

    for f in plan.new + plan.updated:
        film = adb.FilmCreate(
            type=f["type"],
            make=f["make"],
            speed=f["speed"],
            color_type=f["color_type"],
            description=f["description"],
        )
        try:
            analog.upload_film(film)
        except Exception as e:
            failed += 1
            context.log.error(
                f"Failed to upload film, make={film.make}, type={film.type}, speed={film.speed}, {error_detail(e)}"
            )
            continue
        context.log.debug(
            f"Uploaded film, make={film.make}, type={film.type}, speed={film.speed}"
        )

    return upload_result(context, "films", plan, failed)


@dg.asset(group_name="scrape")
def upload_cameras(
    context: dg.AssetExecutionContext,
    cameras_json: CamerasJsonResource,
    analogdb: AnalogDBResource,
) -> dg.MaterializeResult:
    analog = analogdb.client()
    plan = camera_upload_plan(cameras_json.client(), analog.get_cameras())
    failed = 0

    for f in plan.new + plan.updated:
        camera = adb.CameraCreate(
            make=f["make"],
            model=f["model"],
            description=f["description"],
        )
        try:
            analog.upload_camera(camera)
        except Exception as e:
            failed += 1
            context.log.error(
                f"Failed to upload camera, make={camera.make}, model={camera.model}, {error_detail(e)}"
            )
            continue
        context.log.debug(f"Uploaded camera, make={camera.make}, model={camera.model}")

    return upload_result(context, "cameras", plan, failed)


def upload_result(
    context: dg.AssetExecutionContext, kind: str, plan: UploadPlan, failed: int
) -> dg.MaterializeResult:
    """Only new and changed entries are sent: the backend upserts, and every
    conflicting insert still uses up an id."""
    counts = {
        "new": len(plan.new),
        "updated": len(plan.updated),
        "unchanged": plan.unchanged,
        "failed": failed,
        "not_in_json": len(plan.not_in_json),
    }
    context.log.info(f"Uploaded {kind}: {counts}")
    if plan.not_in_json:
        context.log.warning(
            f"Live {kind} missing from the json: {', '.join(plan.not_in_json)}"
        )
    sent = len(plan.new) + len(plan.updated)
    if failed > 0:
        raise dg.Failure(
            description=f"Failed to upload {failed} of {sent} {kind}",
            metadata=counts,
        )
    return dg.MaterializeResult(metadata=counts)
