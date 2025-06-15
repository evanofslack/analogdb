import dagster as dg

from dagster_app.constants import BLACKLIST_PATH, KEYWORD_LIMIT
from .resources import (
    AnalogDBResource,
    ImageProcessorResource,
    KeywordBlacklistResource,
    KeywordExtractorResource,
    MetadataResource,
    RedditResource,
    StorageResource,
)
from .assets import (
    analogdb_permalinks,
    analogdb_posts,
    colors,
    final_posts,
    debug_posts,
    keywords,
    reddit_posts,
    s3_images,
    title_metadatas,
    upload_posts,
)
from dotenv import load_dotenv
from dagster_aws.s3 import S3Resource

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
    },
)
