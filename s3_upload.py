import io
from os import getenv

import boto3
import boto3.session
import requests
from PIL import Image


class UploadError(Exception):
    pass


CLOUDFRONT_URL = "https://d3i73ktnzbi69i.cloudfront.net/"


def init_s3():
    boto_kwargs = {
        "aws_access_key_id": getenv("AWS_ACCESS_KEY_ID"),
        "aws_secret_access_key": getenv("AWS_SECRET_ACCESS_KEY"),
        "region_name": getenv("AWS_REGION"),
    }
    s3 = boto3.client("s3")

    # my_session = boto3.session.Session(**boto_kwargs)
    # s3 = my_session.resource("s3")
    return s3


def s3_upload(
    s3, bucket: str, image: Image.Image, filename: str, content_type: str
) -> str:

    assert bucket == "analog-photos" or bucket == "analog-photos-test"

    in_mem = io.BytesIO()
    image.save(in_mem, content_type.removeprefix("image/"))
    in_mem.seek(0)

    # try:
    # s3.Bucket(bucket).put_object(
    #     Key=filename, Body=in_mem, ContentType=content_type
    # )
    r = s3.upload_fileobj(
        in_mem, bucket, filename, ExtraArgs={"ContentType": content_type}
    )
    print(f"success, uploaded {filename} to {bucket}")
    return CLOUDFRONT_URL + filename

    # except Exception as e:
    #     print(e)
    #     raise UploadError
