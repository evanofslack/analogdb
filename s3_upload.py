from os import getenv
from typing import List

import boto3
import boto3.session
import requests


def init_s3():
    boto_kwargs = {
        "aws_access_key_id": getenv("AWS_ACCESS_KEY_ID"),
        "aws_secret_access_key": getenv("AWS_SECRET_ACCESS_KEY"),
        "region_name": getenv("AWS_REGION"),
    }
    my_session = boto3.session.Session(**boto_kwargs)
    s3 = my_session.resource("s3")
    return s3


def s3_upload(bucket: str, urls: List[str], filenames: List[str]):
    s3 = init_s3()
    assert bucket in [b.name for b in s3.buckets.all()]

    viable_content = {
        "image/png": ".png",
        "image/jpeg": ".jpeg",
        "image/jpg": ".jpg",
        "image/gif": ".gif",
    }

    for url, filename in zip(urls, filenames):

        req = requests.get(url, stream=True)

        content_type = req.headers["content-type"]
        if content_type not in viable_content.keys():
            print(f"Cannot process {url} with type {content_type}")
            pass
        filename += viable_content[content_type]

        req_raw = req.raw
        req_data = req_raw.read()
        s3.Bucket(bucket).put_object(
            Key=filename, Body=req_data, ContentType=content_type
        )
        print(f"success, uploaded {filename} to {bucket}")


if __name__ == "__main__":
    urls = [
        "https://i.redd.it/47fibpjxk8e81.gif",
        "https://i.redd.it/j4hlk7x8e1f81.jpg",
    ]
    filenames = ["3", "4"]
    bucket = "analog-photos"

    s3_upload(bucket, urls, filenames)
