import os

import dagster as dg
from dagster_aws.s3 import S3Resource
from dotenv import load_dotenv

load_dotenv()

from dagster_app.constants import (
    CAMERAS_PATH,
    DEFAULT_VISION_MODEL,
    FILMS_PATH,
    TAG_STOPLIST_PATH,
)
from dagster_app.iomanager import io_manager

from .assets import (
    analogdb_permalinks,
    analogdb_posts,
    debug_posts,
    final_posts,
    patch_post_descriptions,
    patch_post_metadata,
    patch_post_scores,
    post_images,
    post_tags,
    reddit_comments_to_s3,
    reddit_posts,
    title_metadatas,
    updated_post_descriptions,
    updated_post_scores,
    updated_reddit_comments,
    upload_cameras,
    upload_films,
    upload_posts,
)
from .backfill import (
    backfill_post_captions,
    backfill_post_metadata,
    reencode_post_vectors,
    rematch_post_metadata,
)
from .jobs import (
    backfill_captions_job,
    backfill_metadata_job,
    patch_comments_job,
    patch_descriptions_job,
    patch_scores_job,
    reencode_vectors_job,
    rematch_metadata_job,
    scrape_job,
)
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
from .schedules import (
    encode_missing_vectors_schedule,
    scrape_analog_schedule,
    update_post_comments_schedule,
    update_post_scores_schedule,
)

defs = dg.Definitions(
    assets=[
        backfill_post_captions,
        backfill_post_metadata,
        reencode_post_vectors,
        rematch_post_metadata,
        analogdb_permalinks,
        analogdb_posts,
        debug_posts,
        final_posts,
        patch_post_descriptions,
        patch_post_metadata,
        patch_post_scores,
        post_images,
        post_tags,
        reddit_comments_to_s3,
        reddit_posts,
        title_metadatas,
        updated_post_descriptions,
        updated_post_scores,
        updated_reddit_comments,
        upload_cameras,
        upload_films,
        upload_posts,
    ],
    resources={
        "analogdb": AnalogDBResource(
            base_url=dg.EnvVar("ANALOGDB_ENDPOINT"),
            username=dg.EnvVar("ANALOGDB_USERNAME"),
            password=dg.EnvVar("ANALOGDB_PASSWORD"),
            batch_posts_count=dg.EnvVar.int("ANALOGDB_BATCH_POSTS_COUNT"),
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
        "tagger": TaggerResource(
            openai_url=dg.EnvVar("OPENROUTER_BASE_URL"),
            openai_key=dg.EnvVar("OPENROUTER_API_KEY"),
            openai_model=os.environ.get(
                "OPENROUTER_VISION_MODEL", DEFAULT_VISION_MODEL
            ),
            stoplist_path=TAG_STOPLIST_PATH,
        ),
        "films_json": FilmsJsonResource(file_path=FILMS_PATH),
        "cameras_json": CamerasJsonResource(file_path=CAMERAS_PATH),
        "io_manager": io_manager(),
    },
    jobs=[
        backfill_captions_job,
        backfill_metadata_job,
        reencode_vectors_job,
        rematch_metadata_job,
        scrape_job,
        patch_descriptions_job,
        patch_scores_job,
        patch_comments_job,
    ],
    schedules=[
        encode_missing_vectors_schedule,
        scrape_analog_schedule,
        update_post_comments_schedule,
        update_post_scores_schedule,
    ],
)
