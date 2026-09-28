import pickle
from io import BytesIO
from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import numpy as np
import pytest
from dagster_aws.s3 import S3Resource
from PIL import Image
from scrape.models import RedditPost

from .assets import post_images, updated_reddit_comments
from .resources import ImageProcessorResource, RedditResource, StorageResource
from .result import Result, Status


def run_updated_reddit_comments(post_count: int):
    posts = [
        adb.Post(
            id=i,
            title="title",
            author="u/author",
            permalink=f"/r/analog/{i}",
            description=None,
            score=1,
            timestamp=0,
            nsfw=False,
            grayscale=False,
            sprocket=False,
            images=[],
        )
        for i in range(post_count)
    ]
    scraper = MagicMock()
    scraper.scrape_comments.return_value = []
    context = dg.build_asset_context(partition_key="2024-01-01")
    with patch.object(RedditResource, "client", return_value=scraper):
        return updated_reddit_comments(context, posts, RedditResource())


class TestUpdatedRedditComments:
    def test_fails_when_many_posts_have_no_comments(self):
        with pytest.raises(dg.Failure):
            run_updated_reddit_comments(5)

    def test_allows_few_posts_with_no_comments(self):
        result = run_updated_reddit_comments(3)
        assert len(result) == 3


def make_reddit_posts(count: int) -> Result[RedditPost]:
    posts = {}
    for i in range(count):
        permalink = f"https://www.reddit.com/r/analog/comments/{i}/title"
        posts[permalink] = RedditPost(
            image_url=f"https://i.redd.it/{i}.jpg",
            subreddit="analog",
            title="title",
            selftext=None,
            author="u/author",
            permalink=permalink,
            score=1,
            nsfw=False,
            time=0,
            sprocket=False,
        )
    return Result(data=posts, status={id: Status.SUCCESS for id in posts})


def fake_get(bad_urls: set[str]):
    x = np.linspace(0, 255, 1200)[None, :].repeat(900, axis=0)
    pixels = np.stack([x, 255 - x, x / 2], axis=2).astype(np.uint8)
    buf = BytesIO()
    Image.fromarray(pixels).save(buf, "JPEG")
    content = buf.getvalue()

    def get(url, **kwargs):
        resp = MagicMock()
        if url in bad_urls:
            resp.content = b"<html></html>"
            resp.headers = {"content-type": "text/html"}
        else:
            resp.content = content
            resp.headers = {"content-type": "image/jpeg"}
        return resp

    return get


def run_post_images(reddit_posts: Result[RedditPost], bad_urls: set[str]):
    s3_client = MagicMock()
    storage = StorageResource(s3_resource=S3Resource(region_name="us-east-1"))
    context = dg.build_asset_context()
    with (
        patch.object(S3Resource, "get_client", return_value=s3_client),
        patch("scrape.image.requests.get", side_effect=fake_get(bad_urls)),
    ):
        result = post_images(context, ImageProcessorResource(), storage, reddit_posts)
    return result, s3_client


class TestPostImages:
    def test_bad_post_fails_and_rest_succeed(self):
        reddit_posts = make_reddit_posts(3)
        result, s3_client = run_post_images(reddit_posts, {"https://i.redd.it/1.jpg"})

        bad = "https://www.reddit.com/r/analog/comments/1/title"
        assert result.status[bad] == Status.FAILED
        assert "content type" in result.errors[bad]
        assert result.successful_ids() == set(reddit_posts.data) - {bad}
        assert s3_client.put_object.call_count == 8

    def test_output_has_no_images(self):
        result, _ = run_post_images(make_reddit_posts(20), set())

        assert result.successful_count() == 20
        assert len(pickle.dumps(result)) < 100_000
