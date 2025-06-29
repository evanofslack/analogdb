import json
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


@dg.asset
def analogdb_posts(analogdb: AnalogDBResource) -> List[adb.Post]:
    posts = analogdb.client().get_posts_all(count=100)

    dg.get_dagster_logger().info(f"Fetched {len(posts)} posts")

    return posts


@dg.asset
def analogdb_films(analogdb: AnalogDBResource) -> List[adb.Film]:
    films = analogdb.client().get_films()

    dg.get_dagster_logger().info(f"Fetched {len(films)} films")

    return films


@dg.asset
def analogdb_cameras(analogdb: AnalogDBResource) -> List[adb.Camera]:
    cameras = analogdb.client().get_cameras()

    dg.get_dagster_logger().info(f"Fetched {len(cameras)} cameras")

    return cameras


@dg.asset
def analogdb_permalinks(analogdb: AnalogDBResource) -> List[str]:
    links = analogdb.client().get_latest_links(count=100)

    dg.get_dagster_logger().info(f"Fetched {len(links)} post permalinks")

    return links


@dg.asset(dagster_type=ResultDagsterType)
def reddit_posts(
    reddit: RedditResource, analogdb_permalinks: List[str]
) -> Result[RedditPost]:
    result_analog = reddit.client().scrape_posts(
        "analog", 15, analogdb_permalinks, "top"
    )
    dg.get_dagster_logger().info(
        f"Scraped {len(result_analog.posts)} posts from r/analog"
    )

    result_analog_bw = reddit.client().scrape_posts(
        "analog_bw", 2, analogdb_permalinks, "top"
    )
    dg.get_dagster_logger().info(
        f"Scraped {len(result_analog_bw.posts)} posts from r/analog_bw"
    )

    result_sprocket = reddit.client().scrape_posts(
        "SprocketShots", 2, analogdb_permalinks, "top"
    )
    dg.get_dagster_logger().info(
        f"Scraped {len(result_sprocket.posts)} posts from r/sprocketshots"
    )

    posts = result_analog.posts + result_analog_bw.posts + result_sprocket.posts
    errors = result_analog.errors + result_analog_bw.errors + result_sprocket.errors
    for err in errors:
        dg.get_dagster_logger().warn(f"Scrape reddit post, {err}")

    data = {}
    status = {}

    for p in posts:
        id = p.permalink
        data[id] = p
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status)

    dg.get_dagster_logger().info(
        f"Scraped {result.successful_count()} successful posts"
    )

    return result


@dg.asset(dagster_type=ResultDagsterType)
def title_metadatas(
    metadata: MetadataResource,
    reddit_posts,
    analogdb_films: List[adb.Film],
    analogdb_cameras: List[adb.Camera],
) -> Result[PhotoMetadata]:
    posts = [p for _, p in reddit_posts.successful().items()]
    titles = [p.title for p in posts]
    metadatas, prompt = metadata.client().extract(
        titles, analogdb_films, analogdb_cameras
    )

    dg.get_dagster_logger().debug(f"Extract title metadata with prompt:\n{prompt}")
    for t, m in zip(titles, metadatas):
        dg.get_dagster_logger().debug(
            f"Extracted metadata from title, title: {t}, metadata: {m}"
        )

    data = {}
    status = {}

    for m, p in zip(metadatas, posts):
        id = p.permalink
        data[id] = m
        status[id] = Status.SUCCESS

    result = Result(data=data, status=status)

    dg.get_dagster_logger().info(
        f"Extract title metadata from {result.successful_count()} posts"
    )

    return result


@dg.asset(dagster_type=ResultDagsterType)
def s3_images(
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

    dg.get_dagster_logger().info(
        f"Upload s3 images for {result.successful_count()} posts"
    )

    return result


@dg.asset(dagster_type=ResultDagsterType)
def colors(
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

    dg.get_dagster_logger().info(
        f"Extract colors for {result.successful_count()} posts"
    )

    return result


@dg.asset(dagster_type=ResultDagsterType)
def keywords(
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

    dg.get_dagster_logger().info(
        f"Extracted keywords for {result.successful_count()} posts"
    )

    return result


@dg.asset()
def final_posts(
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

    dg.get_dagster_logger().info(f"Creating final posts for {len(ids)} posts")

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
            dg.get_dagster_logger().error(f"Failed to create final post for {id}: {e}")
            status[id] = Status.FAILED
            errors[id] = str(e)

    result = Result(data=data, status=status, errors=errors)
    dg.get_dagster_logger().info(f"Created {result.successful_count()} final posts")

    return result


@dg.asset
def upload_posts(analogdb: AnalogDBResource, final_posts) -> None:
    adb = analogdb.client()
    for _, p in final_posts.successful().items():
        adb.upload_post(convert_create(p))
    dg.get_dagster_logger().info(f"Uploaded {final_posts.successful_count()} posts")


@dg.asset
def updated_post_scores(
    analogdb_posts: List[adb.Post], reddit: RedditResource
) -> List[adb.PostPatch]:
    r = reddit.client()
    patches: List[adb.PostPatch] = []
    for p in analogdb_posts:
        score = r.updated_score(p.permalink, p.score)
        if score is None:
            continue
        patch = adb.create_post_patch(id=p.id, score=score)
        patches.append(patch)
    dg.get_dagster_logger().info(f"Got {len(patches)} updated post scores")
    return patches


@dg.asset
def patch_post_scores(
    updated_post_scores: List[adb.PostPatch], analogdb: AnalogDBResource
) -> None:
    adb = analogdb.client()
    for p in updated_post_scores:
        adb.patch_post(p)
    dg.get_dagster_logger().info(f"Patched {len(updated_post_scores)} post scores")


@dg.asset
def updated_reddit_comments(
    analogdb_posts: List[adb.Post],
    reddit: RedditResource,
) -> List[Tuple[adb.Post, List[RedditComment]]]:
    r = reddit.client()

    post_comments: List[Tuple[adb.Post, List[RedditComment]]] = []
    for p in analogdb_posts:
        c = r.scrape_comments(p.permalink)
        post_comments.append((p, c))
    dg.get_dagster_logger().info(f"Got {len(post_comments)} updated reddit comments")
    return post_comments


@dg.asset
def reddit_comments_to_s3(
    updated_reddit_comments: List[Tuple[adb.Post, List[RedditComment]]],
    keyword_extractor: KeywordExtractorResource,
    storage: StorageResource,
) -> None:
    extractor = keyword_extractor.client()

    for p, c in updated_reddit_comments:
        extractor.upload_s3(p.id, c, storage)
    dg.get_dagster_logger().info(
        f"Upload {len(updated_reddit_comments)} post comments to s3"
    )


@dg.asset
def updated_post_keywords(
    updated_reddit_comments: List[Tuple[adb.Post, List[RedditComment]]],
    keyword_extractor: KeywordExtractorResource,
    keyword_blacklist: KeywordBlacklistResource,
) -> List[adb.PostPatch]:
    kw = keyword_extractor.client()
    blacklist = keyword_blacklist.client().blacklist

    patches: List[adb.PostPatch] = []
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
    dg.get_dagster_logger().info(f"Got {len(patches)} updated post scores")
    return patches


@dg.asset
def patch_post_keywords(
    updated_post_keywords: List[adb.PostPatch], analogdb: AnalogDBResource
) -> None:
    adb = analogdb.client()
    for p in updated_post_keywords:
        adb.patch_post(p)
    dg.get_dagster_logger().info(f"Patched {len(updated_post_keywords)} post keywords")


@dg.asset
def debug_posts(final_posts) -> None:
    """Debug asset to inspect posts instead of uploading"""
    logger = dg.get_dagster_logger()

    logger.info(f"Would upload {final_posts.successful_count()} posts")

    for i, (_, p) in enumerate(final_posts.successful().items()):
        logger.info(f"Post {i+1}: {p.title} by {p.author} with score {p.score}")

    posts_dict = [asdict(p) for _, p in final_posts.successful().items()]
    with open("debug_posts.json", "w") as f:
        json.dump(posts_dict, f, indent=2)

    logger.info("Saved all posts to debug_posts.json")


@dg.asset
def upload_films(films_json: FilmsJsonResource, analogdb: AnalogDBResource) -> None:
    analog = analogdb.client()
    success = 0
    for f in films_json.client():
        film = adb.FilmCreate(
            type=f["type"],
            make=f["make"],
            speed=f["speed"],
            color_type=f["color_type"],
            description=f["description"],
        )

        response = analog.upload_film(film)
        if response.status_code in [200, 201]:
            dg.get_dagster_logger().debug(
                f"Uploaded film: {film.make} {film.type} {film.speed}"
            )
            success += 1
        else:
            dg.get_dagster_logger().warn(f"Fail upload film: {film}")

    dg.get_dagster_logger().info(f"Uploaded {success} films")


@dg.asset
def upload_cameras(
    cameras_json: CamerasJsonResource, analogdb: AnalogDBResource
) -> None:
    analog = analogdb.client()
    success = 0
    for f in cameras_json.client():
        camera = adb.CameraCreate(
            make=f["make"],
            model=f["model"],
            description=f["description"],
        )

        response = analog.upload_camera(camera)
        if response.status_code in [200, 201]:
            dg.get_dagster_logger().debug(
                f"Uploaded camera: {camera.make} {camera.model}"
            )
            success += 1
        else:
            dg.get_dagster_logger().warn(f"Fail upload camera: {camera}")

    dg.get_dagster_logger().info(f"Uploaded {success} camera")
