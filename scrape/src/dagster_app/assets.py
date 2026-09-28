import json
import time
from dataclasses import asdict
from typing import List, Tuple

import analogdb.models as adb
import dagster as dg
from scrape.models import (
    Keyword,
    PhotoMetadata,
    PostImages,
    RedditComment,
    RedditPost,
    new_post_create,
)

from .convert import convert_create
from .resources import (
    AnalogDBResource,
    CamerasJsonResource,
    FilmsJsonResource,
    ImageProcessorResource,
    KeywordBlacklistResource,
    KeywordExtractorResource,
    MetadataResource,
    RedditResource,
    StorageResource,
)
from .result import Result, ResultDagsterType, Status

daily_partitions = dg.TimeWindowPartitionsDefinition(
    start="2018-01-01-00:00",
    cron_schedule="0 0 * * *",
    fmt="%Y-%m-%d",
)

MIN_POSTS_FOR_COMMENTS = 5


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
def analogdb_films(
    context: dg.AssetExecutionContext, analogdb: AnalogDBResource
) -> List[adb.Film]:
    films = analogdb.client().get_films()
    context.log.info(f"Fetched {len(films)} films")
    return films


@dg.asset(group_name="analogdb")
def analogdb_cameras(
    context: dg.AssetExecutionContext, analogdb: AnalogDBResource
) -> List[adb.Camera]:
    cameras = analogdb.client().get_cameras()
    context.log.info(f"Fetched {len(cameras)} cameras")
    return cameras


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
    result_analog = reddit.client().scrape_posts(
        "analog", 15, analogdb_permalinks, "top"
    )
    context.log.info(f"Scraped {len(result_analog.posts)} posts from r/analog")

    result_analog_bw = reddit.client().scrape_posts(
        "analog_bw", 2, analogdb_permalinks, "top"
    )
    context.log.info(f"Scraped {len(result_analog_bw.posts)} posts from r/analog_bw")

    result_sprocket = reddit.client().scrape_posts(
        "SprocketShots", 2, analogdb_permalinks, "top"
    )
    context.log.info(f"Scraped {len(result_sprocket.posts)} posts from r/sprocketshots")

    posts = result_analog.posts + result_analog_bw.posts + result_sprocket.posts
    errors = result_analog.errors + result_analog_bw.errors + result_sprocket.errors
    for err in errors:
        context.log.warn(f"Scrape reddit post, {err}")

    data = {}
    status = {}
    for p in posts:
        id = p.permalink
        data[id] = p
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status)
    context.log.info(f"Scraped {result.successful_count()} successful posts")
    return result


@dg.asset(dagster_type=ResultDagsterType, group_name="scrape")
def title_metadatas(
    context: dg.AssetExecutionContext,
    metadata: MetadataResource,
    reddit_posts,
    analogdb_films: List[adb.Film],
    analogdb_cameras: List[adb.Camera],
) -> Result[PhotoMetadata]:
    posts = [p for _, p in reddit_posts.successful().items()]
    titles = [
        f"title: {p.title}"
        + (f" description: {p.selftext}" if p.selftext is not None else "")
        for p in posts
    ]
    extracted = metadata.client().extract(titles, analogdb_films, analogdb_cameras)
    metadatas = extracted.metadata

    for t, m in zip(titles, metadatas, strict=True):
        context.log.debug(f"Extracted title metadata from {t}, metadata: {m}")

    if extracted.failed > 0:
        context.log.warn(
            f"Failed to extract title metadata for {extracted.failed} of {len(titles)} posts"
        )
    if titles and extracted.failed == len(titles):
        context.log.warn(
            "Failed to extract title metadata for all posts, uploading without metadata"
        )

    data = {}
    status = {}
    for m, p in zip(metadatas, posts, strict=True):
        id = p.permalink
        data[id] = m
        status[id] = Status.SUCCESS

    with_metadata = len([m for m in metadatas if not m.is_empty()])
    context.add_output_metadata(
        {"llm_failed": extracted.failed, "with_metadata": with_metadata}
    )

    result = Result(data=data, status=status)
    context.log.info(f"Extracted title metadata from {result.successful_count()} posts")
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
    context.log.info(f"Processed images for {result.successful_count()} posts")
    return result


@dg.asset(
    dagster_type=ResultDagsterType,
    group_name="scrape",
    retry_policy=dg.RetryPolicy(
        max_retries=2, delay=60, backoff=dg.Backoff.EXPONENTIAL
    ),
)
def keywords(
    context: dg.AssetExecutionContext,
    keyword_extractor: KeywordExtractorResource,
    reddit: RedditResource,
    reddit_posts,
    keyword_blacklist: KeywordBlacklistResource,
) -> Result[Keyword]:
    data = {}
    status = {}
    r = reddit.client()
    kw = keyword_extractor.client()
    blacklist = keyword_blacklist.client().blacklist

    for id, p in reddit_posts.successful().items():
        comments = r.scrape_comments(p.permalink)
        keywords = kw.post_keywords(
            p.title,
            p.score,
            comments,
            keyword_extractor.max_keywords,
            blacklist,
        )
        id = p.permalink
        data[id] = keywords
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status)
    context.log.info(f"Extracted keywords for {result.successful_count()} posts")
    return result


@dg.asset(group_name="scrape")
def final_posts(
    context: dg.AssetExecutionContext,
    reddit_posts,
    title_metadatas,
    post_images,
    keywords,
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

    for id in ids:
        try:
            final = new_post_create(
                post=reddit_posts.data[id],
                metadata=title_metadatas.data[id],
                images=post_images.data[id],
                keywords=keywords.data[id],
            )

            data[id] = final
            status[id] = Status.SUCCESS

        except Exception as e:
            context.log.error(f"Failed to create final post for {id}: {e}")
            status[id] = Status.FAILED
            errors[id] = str(e)

    result = Result(data=data, status=status, errors=errors)
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
) -> None:
    adb = analogdb.client()
    for _, p in final_posts.successful().items():
        adb.upload_post(convert_create(p))
    context.log.info(f"Uploaded {final_posts.successful_count()} posts")


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

    adb = analogdb.client()
    for id, p in updated_post_scores:
        adb.patch_post(id, p)
    context.log.info(
        f"Patched {len(updated_post_scores)} post scores for partition {context.partition_key}"
    )


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

    adb = analogdb.client()
    for id, p in updated_post_descriptions:
        adb.patch_post(id, p)
    context.log.info(
        f"Patched {len(updated_post_descriptions)} post descriptions for partition {context.partition_key}"
    )


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def updated_post_title_metadatas(
    context: dg.AssetExecutionContext,
    analogdb_posts: List[adb.Post],
    metadata: MetadataResource,
    analogdb_films: List[adb.Film],
    analogdb_cameras: List[adb.Camera],
) -> List[Tuple[int, adb.PostPatch]]:
    patches: List[Tuple[int, adb.PostPatch]] = []

    if not analogdb_posts:
        context.log.info(f"No posts to process for partition {context.partition_key}")
        return patches

    titles = [
        f"title: {p.title}"
        + (f" description: {p.description}" if p.description is not None else "")
        for p in analogdb_posts
    ]
    extracted = metadata.client().extract(titles, analogdb_films, analogdb_cameras)
    if extracted.failed > 0:
        context.log.warn(
            f"Failed to extract title metadata for {extracted.failed} of {len(titles)} posts"
        )

    for p, m in zip(analogdb_posts, extracted.metadata, strict=True):
        if m.is_empty():
            context.log.debug(
                f"Skip create patch for empty post title metadata, title={p.title}"
            )
            continue
        patch = adb.PostPatch(
            camera_make=m.camera_make,
            camera_model=m.camera_model,
            film_make=m.film_make,
            film_type=m.film_type,
            film_speed=m.film_speed,
            focal_length=m.focal_length,
            aperture=m.aperture,
        )
        context.log.debug(
            f"Created patch for post title metadata, title={p.title}, description={p.description if p.description is not None else ""}, metadata={patch}"
        )
        patches.append((p.id, patch))

    context.log.info(f"Created {len(patches)} updated post title metadatas")
    return patches


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_title_metadatas(
    context: dg.AssetExecutionContext,
    updated_post_title_metadatas: List[Tuple[int, adb.PostPatch]],
    analogdb: AnalogDBResource,
) -> None:
    if not updated_post_title_metadatas:
        context.log.info(f"No patches to apply for partition {context.partition_key}")
        return

    adb = analogdb.client()
    for id, p in updated_post_title_metadatas:
        adb.patch_post(id, p)
        time.sleep(0.2)

    context.log.info(
        f"Patched {len(updated_post_title_metadatas)} post title metadatas for partition {context.partition_key}"
    )


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
def reddit_comments_to_s3(
    context: dg.AssetExecutionContext,
    updated_reddit_comments: List[Tuple[adb.Post, List[RedditComment]]],
    keyword_extractor: KeywordExtractorResource,
    storage: StorageResource,
) -> None:
    if not updated_reddit_comments:
        context.log.info(
            f"No reddit comments to s3 for partition {context.partition_key}"
        )
        return

    extractor = keyword_extractor.client()

    for p, c in updated_reddit_comments:
        extractor.upload_s3(p.id, c, storage)

    context.log.info(
        f"Uploaded {len(updated_reddit_comments)} reddit comments to s3 for partition {context.partition_key}"
    )


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def updated_post_keywords(
    context: dg.AssetExecutionContext,
    updated_reddit_comments: List[Tuple[adb.Post, List[RedditComment]]],
    keyword_extractor: KeywordExtractorResource,
    keyword_blacklist: KeywordBlacklistResource,
) -> List[Tuple[int, adb.PostPatch]]:
    patches: List[Tuple[int, adb.PostPatch]] = []
    if not updated_reddit_comments:
        context.log.info(
            f"No updated post keywords for partition {context.partition_key}"
        )
        return patches

    kw = keyword_extractor.client()
    blacklist = keyword_blacklist.client().blacklist

    for p, c in updated_reddit_comments:
        adb_kws: List[adb.Keyword] = []
        keywords = kw.post_keywords(
            p.title,
            p.score,
            c,
            keyword_extractor.max_keywords,
            blacklist,
        )
        for k in keywords:
            adb_kws.append(adb.Keyword(word=k.word, weight=k.weight))
        patches.append((p.id, adb.PostPatch(keywords=adb_kws)))

    context.log.info(
        f"Created {len(patches)} updated post keywords for partition {context.partition_key}"
    )
    return patches


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_keywords(
    context: dg.AssetExecutionContext,
    updated_post_keywords: List[Tuple[int, adb.PostPatch]],
    analogdb: AnalogDBResource,
) -> None:
    if not updated_post_keywords:
        context.log.info(
            f"No patch post keywords for partition {context.partition_key}"
        )
        return

    adb = analogdb.client()
    for id, p in updated_post_keywords:
        adb.patch_post(id, p)
    context.log.info(f"Patched {len(updated_post_keywords)} post keywords")
    context.log.info(
        f"Patched {len(updated_post_keywords)} post keywords for partition {context.partition_key}"
    )


@dg.asset(group_name="scrape")
def debug_posts(context: dg.AssetExecutionContext, final_posts) -> None:
    """Debug asset to inspect posts instead of uploading"""
    logger = context.log

    logger.info(f"Would upload {final_posts.successful_count()} posts")

    for i, (_, p) in enumerate(final_posts.successful().items()):
        logger.info(f"Post {i+1}: {p.title} by {p.author} with score {p.score}")

    posts_dict = [asdict(p) for _, p in final_posts.successful().items()]
    with open("debug_posts.json", "w") as f:
        json.dump(posts_dict, f, indent=2)

    logger.info("Saved all posts to debug_posts.json")


@dg.asset(group_name="scrape")
def upload_films(
    context: dg.AssetExecutionContext,
    films_json: FilmsJsonResource,
    analogdb: AnalogDBResource,
) -> None:
    analog = analogdb.client()
    success = 0
    max_retries = 5

    for f in films_json.client():
        film = adb.FilmCreate(
            type=f["type"],
            make=f["make"],
            speed=f["speed"],
            color_type=f["color_type"],
            description=f["description"],
        )

        for attempt in range(max_retries):
            resp = analog.upload_film(film)
            if resp.status_code in [200, 201]:
                context.log.debug(
                    f"Uploaded film, make={film.make}, type={film.type}, speed={film.speed}"
                )
                success += 1
                break
            elif attempt == max_retries - 1:
                context.log.warn(
                    f"Fail upload film, attempt={attempt+1}, max_retries={max_retries}, make={film.make}, type={film.type}, speed={film.speed}, body={resp.text}, status={resp.status_code}"
                )
            else:
                context.log.debug(
                    f"Retry upload film, attempt={attempt+1}, max_retries={max_retries}, make={film.make}, type={film.type}, speed={film.speed}, body={resp.text}, status={resp.status_code}"
                )

    context.log.info(f"Uploaded {success} films")


@dg.asset(group_name="scrape")
def upload_cameras(
    context: dg.AssetExecutionContext,
    cameras_json: CamerasJsonResource,
    analogdb: AnalogDBResource,
) -> None:
    analog = analogdb.client()
    success = 0
    max_retries = 5

    for f in cameras_json.client():
        camera = adb.CameraCreate(
            make=f["make"],
            model=f["model"],
            description=f["description"],
        )

        for attempt in range(max_retries):
            resp = analog.upload_camera(camera)
            if resp.status_code in [200, 201]:
                context.log.debug(
                    f"Uploaded camera, make={camera.make}, model={camera.model}"
                )
                success += 1
                break
            elif attempt == max_retries - 1:
                context.log.warn(
                    f"Fail upload camera attempt={attempt+1}, max_retries={max_retries}, make={camera.make}, model={camera.model}, body={resp.text}, status={resp.status_code}"
                )
            else:
                context.log.debug(
                    f"Retry upload camera, attempt={attempt+1}, max_retries={max_retries}, make={camera.make}, model={camera.model}, body={resp.text}, status={resp.status_code}"
                )

    context.log.info(f"Uploaded {success} cameras")
