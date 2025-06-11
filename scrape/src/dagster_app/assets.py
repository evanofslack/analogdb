import dagster as dg
from analogdb.models import Post
from scrape.models import RedditPost, UploadPost
from typing import List
from .resources import (
    AnalogDBResource,
    ImageProcessorResource,
    MetadataResource,
    RedditResource,
    StorageResource,
)
from .result import RedditPosts, Status, TitleMetadatas, S3Images


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
def combine(
    reddit_posts: RedditPosts, title_metadatas: TitleMetadatas, s3_images: S3Images
) -> UploadPosts:
    return


@dg.asset
def upload_posts(analogdb: AnalogDBResource, hydrated_posts: List[UploadPost]):
    for p in hydrated_posts:
        analogdb.upload(p)
    dg.get_dagster_logger().info(f"Uploaded {len(hydrated_posts)} posts")


# def create_upload_post(
#     reddit: RedditPost, metadata: PhotoMetadata, keywords: List[str]
# ) -> UploadPost:
#     # up := UploadPost(url=reddit.permalink, title=reddit.title, author=reddit.author, permalink=reddit.permalink, score=reddit.score, nsfw=reddit.nsfw, grayscale=reddit.grayscale, time=reddit.time, width=reddit.width, height=reddit.height, sprocket=reddit.sprocket, low_url=)
#
#     low_img = images[0]
#     med_img = images[1]
#     high_img = images[2]
#     raw_img = images[3]
#
#     up = UploadPost(
#         url=raw_img.url,
#         title=post.title,
#         author=post.author,
#         permalink=post.permalink,
#         score=post.score,
#         nsfw=post.nsfw,
#         grayscale=post.greyscale,
#         time=post.time,
#         width=raw_img.width,
#         height=raw_img.height,
#         sprocket=post.sprocket,
#         low_url=low_img.url,
#         low_width=low_img.width,
#         low_height=low_img.height,
#         med_url=med_img.url,
#         med_width=med_img.width,
#         med_height=med_img.height,
#         high_url=high_img.url,
#         high_width=high_img.width,
#         high_height=high_img.height,
#         camera_make=metadata.camera_make,
#         camera_model=metadata.camera_model,
#         film_make=metadata.film_make,
#         film_type=metadata.film_type,
#         film_speed=metadata.film_speed,
#         focal_length=metadata.focal_length,
#         aperture=metadata.aperture,
#         keywords=keywords,
#         colors=colors,
#     )
#     return up
