import json
from dataclasses import asdict
from typing import List

import dagster as dg
from analogdb.models import Post, PostPatch, create_post_patch
from scrape.models import (
    Color,
    Keyword,
    PhotoMetadata,
    RedditPost,
    S3Image,
    UploadPost,
    create_upload_post,
)

from .resources import (
    AnalogDBResource,
    ImageProcessorResource,
    KeywordBlacklistResource,
    KeywordExtractorResource,
    MetadataResource,
    RedditResource,
    StorageResource,
)
from .result import Result, ResultDagsterType, Status


@dg.asset
def analogdb_posts(analogdb: AnalogDBResource) -> List[Post]:
    posts = analogdb.client().get_posts_all(count=100)

    dg.get_dagster_logger().info(f"Fetched {len(posts)} posts")

    return posts


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
        f"Scraped {len(result_analog.posts)} posts from r/analog_bw"
    )

    result_sprocket = reddit.client().scrape_posts(
        "SprocketShots", 2, analogdb_permalinks, "top"
    )
    dg.get_dagster_logger().info(
        f"Scraped {len(result_analog.posts)} posts from r/sprocketshots"
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
) -> Result[PhotoMetadata]:
    posts = [p for _, p in reddit_posts.successful().items()]
    metadatas = metadata.client().extract([p.title for p in posts])

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

    for id, p in reddit_posts.successful().items():
        comments = reddit.client().scrape_comments(p.permalink)
        keywords = keyword_extractor.client().post_keywords(
            p.title,
            p.score,
            comments,
            keyword_extractor.max_keywords,
            keyword_blacklist.client().blacklist,
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
            final = create_upload_post(
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
    for _, p in final_posts.successful().items():
        analogdb.upload(p)
    dg.get_dagster_logger().info(f"Uploaded {len(final_posts)} posts")


@dg.asset
def updated_posts_scores(
    analogdb_posts: List[Post], reddit: RedditResource
) -> List[PostPatch]:
    r = reddit.client()
    patches: List[PostPatch] = []
    for p in analogdb_posts:
        score = r.updated_score(p.permalink, p.score)
        if score is None:
            continue
        patch = create_post_patch(id=p.id, score=score)
        patches.append(patch)
    dg.get_dagster_logger().info(f"Got {len(patches)} updated post scores")
    return patches


@dg.asset
def patch_posts_scores(
    updated_posts_scores: List[PostPatch], analogdb: AnalogDBResource
) -> None:
    adb = analogdb.client()
    for p in updated_posts_scores:
        adb.patch_post(p)
    dg.get_dagster_logger().info(f"Patched {len(updated_posts_scores)} post scores")


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
