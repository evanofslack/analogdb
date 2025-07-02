import os

from dagster import EnvVar
from dagster_aws.s3 import S3PickleIOManager, S3Resource

s3_io_manager = S3PickleIOManager(
    s3_resource=S3Resource(
        endpoint_url=EnvVar("DAGSTER_S3_ENDPOINT_URL"),
        aws_access_key_id=EnvVar("DAGSTER_S3_ACCESS_KEY"),
        aws_secret_access_key=EnvVar("DAGSTER_S3_SECRET_KEY"),
    ),
    s3_bucket=EnvVar("DAGSTER_S3_BUCKET"),
    s3_prefix=EnvVar("DAGSTER_S3_PREFIX"),
)


def io_manager():
    env = os.getenv("ANALOGDB_APP_ENV", "development").lower()

    if env == "production" or env == "prod":
        return s3_io_manager
    else:
        from dagster import FilesystemIOManager

        return FilesystemIOManager(base_dir="./dagster_storage")
