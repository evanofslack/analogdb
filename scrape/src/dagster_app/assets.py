import dagster as dg
import json
from analogdb.models import Post
from dataclasses import asdict
from scrape.models import (
    RedditPost,
    UploadPost,
    create_upload_post,
)
from typing import List
from .resources import (
    AnalogDBResource,
    ImageProcessorResource,
    KeywordBlacklistResource,
    KeywordExtractorResource,
    MetadataResource,
    RedditResource,
    StorageResource,
)
from .result import (
    Colors,
    Keywords,
    RedditPosts,
    Status,
    TitleMetadatas,
    S3Images,
    FinalPosts,
)


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


@dg.asset
def reddit_posts(reddit: RedditResource, analogdb_permalinks: List[str]) -> RedditPosts:
    api_result = reddit.client().scrape_posts("analog", 100, analogdb_permalinks, "hot")
    for err in api_result.errors:
        dg.get_dagster_logger().warn(f"Scrape reddit post, {err}")

    data = {}
    status = {}

    for post in api_result.posts:
        post_id = post.permalink
        data[post_id] = post
        status[post_id] = Status.SUCCESS

    result = RedditPosts(data=data, status=status)

    dg.get_dagster_logger().info(f"Scraped {result.successful_count()} posts")

    return result


@dg.asset
def title_metadata(
    extractor: MetadataResource, reddit_posts: List[RedditPost]
) -> TitleMetadatas:
    metadatas = extractor.client().extract([p.title for p in reddit_posts])

    data = {}
    status = {}

    for m, p in zip(metadatas, reddit_posts):
        id = p.permalink
        data[id] = m
        status[id] = Status.SUCCESS

    result = TitleMetadatas(data=data, status=status)

    dg.get_dagster_logger().info(
        f"Extract title metadata from {result.successful_count()} posts"
    )

    return result


@dg.asset
def s3_images(
    processor: ImageProcessorResource,
    s3: StorageResource,
    reddit_posts: List[RedditPost],
) -> S3Images:
    data = {}
    status = {}

    for p in reddit_posts:
        images = processor.client().upload_s3(p, s3)
        id = p.permalink
        data[id] = images
        status[id] = Status.SUCCESS

    result = S3Images(data=data, status=status)

    dg.get_dagster_logger().info(
        f"Upload s3 images for {result.successful_count()} posts"
    )

    return result


@dg.asset
def colors(
    processor: ImageProcessorResource,
    reddit_posts: List[RedditPost],
) -> Colors:
    data = {}
    status = {}

    for p in reddit_posts:
        colors = processor.client().extract_colors(p.image)
        id = p.permalink
        data[id] = colors
        status[id] = Status.SUCCESS

    result = Colors(data=data, status=status)

    dg.get_dagster_logger().info(
        f"Extract colors for {result.successful_count()} posts"
    )

    return result


@dg.asset
def keywords(
    extractor: KeywordExtractorResource,
    scraper: RedditResource,
    reddit_posts: List[RedditPost],
    blacklist: KeywordBlacklistResource,
) -> Keywords:
    data = {}
    status = {}

    for p in reddit_posts:
        comments = scraper.client().scrape_comments(p.permalink)
        keywords = extractor.client().post_keywords(
            p.title, p.score, comments, extractor.max_keywords, blacklist.blacklist
        )
        id = p.permalink
        data[id] = keywords
        status[id] = Status.SUCCESS

    result = Keywords(data=data, status=status)

    dg.get_dagster_logger().info(
        f"Extracte keywords for {result.successful_count()} posts"
    )

    return result


@dg.asset
def combine(
    reddit_posts: RedditPosts,
    title_metadatas: TitleMetadatas,
    s3_images: S3Images,
    colors: Colors,
    keywords: Keywords,
) -> FinalPosts:
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

    result = FinalPosts(data=data, status=status, errors=errors)
    dg.get_dagster_logger().info(f"Created {result.successful_count()} final posts")

    return result


@dg.asset
def upload_posts(analogdb: AnalogDBResource, final_posts: List[UploadPost]):
    for p in final_posts:
        analogdb.upload(p)
    dg.get_dagster_logger().info(f"Uploaded {len(final_posts)} posts")


@dg.asset
def debug_posts(final_posts: List[UploadPost]) -> None:
    """Debug asset to inspect posts instead of uploading"""
    logger = dg.get_dagster_logger()

    logger.info(f"Would upload {len(final_posts)} posts")

    for i, p in enumerate(final_posts):
        logger.info(f"Post {i+1}: {p.title} by {p.author} with score {p.score}")

    posts_dict = [asdict(p) for p in final_posts]
    with open("debug_posts.json", "w") as f:
        json.dump(posts_dict, f, indent=2)

    logger.info("Saved all posts to debug_posts.json")
