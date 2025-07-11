import json
import time
from dataclasses import asdict
from typing import List, Tuple

import analogdb.models as adb
import dagster as dg
from scrape.models import (
    Color,
    Keyword,
    PhotoMetadata,
    RedditComment,
    RedditPost,
    S3Image,
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


@dg.asset(partitions_def=daily_partitions, group_name="analogdb")
def analogdb_posts(
    context: dg.AssetExecutionContext, analogdb: AnalogDBResource
) -> List[adb.Post]:
    window = context.partition_time_window
    time_start = int(window.start.timestamp())
    time_end = int(window.end.timestamp())

    filter = adb.create_posts_filter(time_start=time_start, time_end=time_end)
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


@dg.asset(dagster_type=ResultDagsterType, group_name="scrape")
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
    titles = [f"title: {p.title} description: {p.selftext}" for p in posts]
    metadatas, _ = metadata.client().extract(titles, analogdb_films, analogdb_cameras)

    # context.log.debug(f"Extract title metadata with prompt:\n{prompt}")
    for t, m in zip(titles, metadatas):
        context.log.debug(f"Extracted title metadata from {t}, metadata: {m}")

    data = {}
    status = {}
    for m, p in zip(metadatas, posts):
        id = p.permalink
        data[id] = m
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status)
    context.log.info(f"Extracted title metadata from {result.successful_count()} posts")
    return result


@dg.asset(dagster_type=ResultDagsterType, group_name="scrape")
def s3_images(
    context: dg.AssetExecutionContext,
    image_processor: ImageProcessorResource,
    storage: StorageResource,
    reddit_posts,
) -> Result[List[S3Image]]:
    data = {}
    status = {}
    for _, p in reddit_posts.successful().items():
        images = image_processor.client().upload_s3(p, storage)
        id = p.permalink
        data[id] = images
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status)
    context.log.info(f"Uploaded s3 images for {result.successful_count()} posts")
    return result


@dg.asset(dagster_type=ResultDagsterType, group_name="scrape")
def colors(
    context: dg.AssetExecutionContext,
    image_processor: ImageProcessorResource,
    reddit_posts,
) -> Result[Color]:
    data = {}
    status = {}
    for _, p in reddit_posts.successful().items():
        colors = image_processor.client().extract_colors(p.image)
        id = p.permalink
        data[id] = colors
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status)
    context.log.info(f"Extracted colors for {result.successful_count()} posts")
    return result


@dg.asset(dagster_type=ResultDagsterType, group_name="scrape")
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
    s3_images,
    colors,
    keywords,
):
    ids = (
        reddit_posts.successful_ids()
        & title_metadatas.successful_ids()
        & s3_images.successful_ids()
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
                images=s3_images.data[id],
                keywords=keywords.data[id],
                colors=colors.data[id],
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


@dg.asset(group_name="scrape")
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
) -> List[adb.PostPatch]:
    r = reddit.client()
    patches: List[adb.PostPatch] = []
    for p in analogdb_posts:
        score = r.updated_score(p.permalink, p.score)
        if score is None:
            continue
        patch = adb.create_post_patch(id=p.id, score=score)
        patches.append(patch)
    context.log.info(f"Created {len(patches)} updated post scores")
    return patches


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_scores(
    context: dg.AssetExecutionContext,
    updated_post_scores: List[adb.PostPatch],
    analogdb: AnalogDBResource,
) -> None:
    if not updated_post_scores:
        context.log.info(
            f"No updated post scores to process for partition {context.partition_key}"
        )

    adb = analogdb.client()
    for p in updated_post_scores:
        adb.patch_post(p)
    context.log.info(
        f"Patched {len(updated_post_scores)} post scores for partition {context.partition_key}"
    )


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def updated_post_title_metadatas(
    context: dg.AssetExecutionContext,
    analogdb_posts: List[adb.Post],
    metadata: MetadataResource,
    analogdb_films: List[adb.Film],
    analogdb_cameras: List[adb.Camera],
) -> List[adb.PostPatch]:
    patches: List[adb.PostPatch] = []

    if not analogdb_posts:
        context.log.info(f"No posts to process for partition {context.partition_key}")
        return patches

    titles = [p.title for p in analogdb_posts]
    metadatas, _ = metadata.client().extract(titles, analogdb_films, analogdb_cameras)
    if len(analogdb_posts) != len(metadatas):
        context.log.error(
            f"Unequal count of posts and extracted metadata, {len(analogdb_posts)} != {len(metadatas)}"
        )
        raise Exception("Unequal count of posts and extracted metadata")

    for p, m in zip(analogdb_posts, metadatas):
        meta = adb.PhotoMetadata(
            m.camera_make,
            m.camera_model,
            m.film_make,
            m.film_type,
            m.film_speed,
            m.focal_length,
            m.aperture,
        )
        if meta.is_empty() or m.is_empty():
            context.log.debug(
                f"Skip create patch for empty post title metadata, title={p.title}, empty={meta.is_empty()}, empty={m.is_empty()}, metadata={meta}"
            )
            continue
        context.log.debug(
            f"Created patch for post title metadata, title={p.title}, metadata={meta}"
        )
        patch = adb.create_post_patch(id=p.id, metadata=meta)
        patches.append(patch)

    context.log.info(f"Created {len(patches)} updated post title metadatas")
    return patches


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_title_metadatas(
    context: dg.AssetExecutionContext,
    updated_post_title_metadatas: List[adb.PostPatch],
    analogdb: AnalogDBResource,
) -> None:
    if not updated_post_title_metadatas:
        context.log.info(f"No patches to apply for partition {context.partition_key}")
        return

    adb = analogdb.client()
    for p in updated_post_title_metadatas:
        adb.patch_post(p)
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
) -> List[adb.PostPatch]:
    patches: List[adb.PostPatch] = []
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
            adb_kws.append(adb.Keyword(k.word, k.weight))
        patch = adb.create_post_patch(id=p.id, keywords=adb_kws)
        patches.append(patch)

    context.log.info(
        f"Created {len(patches)} updated post keywords for partition {context.partition_key}"
    )
    return patches


@dg.asset(partitions_def=daily_partitions, group_name="backfill")
def patch_post_keywords(
    context: dg.AssetExecutionContext,
    updated_post_keywords: List[adb.PostPatch],
    analogdb: AnalogDBResource,
) -> None:
    if not updated_post_keywords:
        context.log.info(
            f"No patch post keywords for partition {context.partition_key}"
        )
        return

    adb = analogdb.client()
    for p in updated_post_keywords:
        adb.patch_post(p)
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
