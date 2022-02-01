from os import getenv
from typing import List

import boto3
import boto3.session
import requests


class UploadError(Exception):
    pass


CLOUDFRONT_URL = "https://d3i73ktnzbi69i.cloudfront.net/"


def init_s3():
    boto_kwargs = {
        "aws_access_key_id": getenv("AWS_ACCESS_KEY_ID"),
        "aws_secret_access_key": getenv("AWS_SECRET_ACCESS_KEY"),
        "region_name": getenv("AWS_REGION"),
    }
    my_session = boto3.session.Session(**boto_kwargs)
    s3 = my_session.resource("s3")
    return s3


def s3_upload(s3, bucket: str, url: str, filename: str) -> str:

    assert bucket in [b.name for b in s3.buckets.all()]

    viable_content = {
        "image/png": ".png",
        "image/jpeg": ".jpeg",
        "image/jpg": ".jpg",
        "image/gif": ".gif",
    }

    try:
        req = requests.get(url, stream=True)
    except Exception as e:
        print(e)
        raise UploadError

    content_type = req.headers["content-type"]
    if content_type not in viable_content.keys():
        print(f"Cannot process {url} with type {content_type}")
        raise UploadError
    filename += viable_content[content_type]

    try:
        req_raw = req.raw
        req_data = req_raw.read()
        s3.Bucket(bucket).put_object(
            Key=filename, Body=req_data, ContentType=content_type
        )
        print(f"success, uploaded {filename} to {bucket}")
        return CLOUDFRONT_URL + filename

    except Exception as e:
        print(e)
        raise UploadError


if __name__ == "__main__":
    url = "https://i.redd.it/sgbeui51l1f81.jpg"
    filename = "6"
    bucket = "analog-photos"

    s3 = init_s3()
    upload_url = s3_upload(s3, bucket, url, filename)
    print(upload_url)
