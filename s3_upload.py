import io

import boto3
import boto3.session
from PIL import Image

from configuration import Config
from constants import AWS_BUCKET, AWS_BUCKET_TEST, CLOUDFRONT_URL


def init_s3(config: Config) -> boto3.session.Session:
    s3 = boto3.client(
        "s3",
        aws_access_key_id=config.aws.access_key_id,
        aws_secret_access_key=config.aws.secret_access_key,
        region_name=config.aws.region_name,
    )
    return s3


def s3_upload(
    s3, bucket: str, image: Image.Image, filename: str, content_type: str
) -> str:
    assert bucket == AWS_BUCKET or bucket == AWS_BUCKET_TEST

    in_mem = io.BytesIO()
    image.save(in_mem, content_type.removeprefix("image/"))
    in_mem.seek(0)
    s3.upload_fileobj(in_mem, bucket, filename, ExtraArgs={"ContentType": content_type})

    print(f"success, uploaded {filename} to {bucket}")
    return CLOUDFRONT_URL + filename
