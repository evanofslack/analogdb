import uuid
from typing import List, Tuple

import boto3
import boto3.session
from loguru import logger
from PIL.Image import Image

from constants import (AWS_BUCKET, AWS_BUCKET_TEST, CLOUDFRONT_URL, HIGH_RES,
                       LOW_RES, MEDIUM_RES, RAW_RES)
from image_process import extract_colors, image_to_bytes, resize_image
from models import AnalogPost, CloudfrontImage, RedditPost


def create_filename(content_type: str) -> str:
    content_suffix = {
        "image/png": "png",
        "image/jpeg": "jpeg",
        "image/jpg": "jpg",
        "image/gif": "gif",
    }

    _uuid = str(uuid.uuid4())
    suffix = content_suffix[content_type]
    filename = f"{_uuid}.{suffix}"
    return filename


def upload_image_to_s3(
    s3, bucket: str, image: Image, filename: str, content_type: str
) -> str:
    assert bucket == AWS_BUCKET or bucket == AWS_BUCKET_TEST

    img_bytes = image_to_bytes(image=image, content_type=content_type)

    try:
        s3.upload_fileobj(
            img_bytes, bucket, filename, ExtraArgs={"ContentType": content_type}
        )
    except Exception as e:
        logger.error(f"failed to upload {filename} to {bucket} with error: {e}")
        raise e

    url = f"{CLOUDFRONT_URL}/{filename}"
    logger.info(f"uploaded to {bucket} ({url})")
    return url


def upload_to_s3(
    post: RedditPost, s3: boto3.session.Session, bucket: str
) -> List[CloudfrontImage]:

    cf_images: List[CloudfrontImage] = []
    resolutions: List[Tuple[int, int]] = [LOW_RES, MEDIUM_RES, HIGH_RES, RAW_RES]
    for res in resolutions:

        image, width, height = resize_image(image=post.image, size=res)
        filename = create_filename(content_type=post.content_type)

        url = upload_image_to_s3(
            s3=s3,
            bucket=bucket,
            image=image,
            filename=filename,
            content_type=post.content_type,
        )

        cf_image = CloudfrontImage(url=url, width=width, height=height)
        cf_images.append(cf_image)

    return cf_images


def create_analog_post(images: List[CloudfrontImage], post: RedditPost) -> AnalogPost:

    low_img = images[0]
    med_img = images[1]
    high_img = images[2]
    raw_img = images[3]

    colors = extract_colors(image=post.image)
    c1 = colors[0]
    c2 = colors[1]
    c3 = colors[2]
    c4 = colors[3]
    c5 = colors[4]

    analog_post = AnalogPost(
        url=raw_img.url,
        title=post.title,
        author=post.author,
        permalink=post.permalink,
        score=post.score,
        nsfw=post.nsfw,
        greyscale=post.greyscale,
        time=post.time,
        width=raw_img.width,
        height=raw_img.height,
        sprocket=post.sprocket,
        low_url=low_img.url,
        low_width=low_img.width,
        low_height=low_img.height,
        med_url=med_img.url,
        med_width=med_img.width,
        med_height=med_img.height,
        high_url=high_img.url,
        high_width=high_img.width,
        high_height=high_img.height,
        c1_hex=c1.hex,
        c1_css=c1.css,
        c1_percent=c1.percent,
        c2_hex=c2.hex,
        c2_css=c2.css,
        c2_percent=c2.percent,
        c3_hex=c3.hex,
        c3_css=c3.css,
        c3_percent=c3.percent,
        c4_hex=c4.hex,
        c4_css=c4.css,
        c4_percent=c4.percent,
        c5_hex=c5.hex,
        c5_css=c5.css,
        c5_percent=c5.percent,
    )

    return analog_post
