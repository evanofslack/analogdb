import dagster as dg
from dagster_aws.s3 import S3Resource
from dotenv import load_dotenv

from dagster_app.constants import (
    BLACKLIST_PATH,
    CAMERAS_PATH,
    FILMS_PATH,
    KEYWORD_LIMIT,
)

from .assets import (
    analogdb_permalinks,
    analogdb_posts,
    colors,
    debug_posts,
    final_posts,
    keywords,
    patch_post_keywords,
    patch_post_scores,
    reddit_comments_to_s3,
    reddit_posts,
    s3_images,
    title_metadatas,
    updated_post_keywords,
    updated_post_scores,
    updated_reddit_comments,
    upload_cameras,
    upload_films,
    upload_posts,
)
from .jobs import patch_keywords_job, patch_scores_job, scrape_job
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
from .schedules import scrape_analog_schedule

load_dotenv()

defs = dg.Definitions(
    assets=[
        analogdb_posts,
        analogdb_permalinks,
        reddit_posts,
        title_metadatas,
        s3_images,
        colors,
        keywords,
        final_posts,
        upload_posts,
        debug_posts,
        updated_post_scores,
        patch_post_scores,
        updated_reddit_comments,
        reddit_comments_to_s3,
        updated_post_keywords,
        patch_post_keywords,
        upload_films,
        upload_cameras,
    ],
    resources={
        "analogdb": AnalogDBResource(
            base_url=dg.EnvVar("ANALOGDB_ENDPOINT"),
            username=dg.EnvVar("ANALOGDB_USERNAME"),
            password=dg.EnvVar("ANALOGDB_PASSWORD"),
        ),
        "reddit": RedditResource(
            client_id=dg.EnvVar("REDDIT_CLIENT_ID"),
            client_secret=dg.EnvVar("REDDIT_CLIENT_SECRET"),
            user_agent=dg.EnvVar("REDDIT_USER_AGENT"),
        ),
        "image_processor": ImageProcessorResource(),
        "metadata": MetadataResource(
            openai_url=dg.EnvVar("OPENROUTER_BASE_URL"),
            openai_key=dg.EnvVar("OPENROUTER_API_KEY"),
            openai_model=dg.EnvVar("OPENROUTER_MODEL"),
        ),
        "storage": StorageResource(
            s3_resource=S3Resource(
                aws_access_key_id=dg.EnvVar("AWS_ACCESS_KEY_ID"),
                aws_secret_access_key=dg.EnvVar("AWS_SECRET_ACCESS_KEY"),
                region_name=dg.EnvVar("AWS_REGION"),
            )
        ),
        "keyword_extractor": KeywordExtractorResource(max_keywords=KEYWORD_LIMIT),
        "keyword_blacklist": KeywordBlacklistResource(file_path=BLACKLIST_PATH),
        "films_json": FilmsJsonResource(file_path=FILMS_PATH),
        "cameras_json": CamerasJsonResource(file_path=CAMERAS_PATH),
    },
    jobs=[scrape_job, patch_scores_job, patch_keywords_job],
    schedules=[scrape_analog_schedule],
)
