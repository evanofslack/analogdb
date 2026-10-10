import pickle
from io import BytesIO
from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import numpy as np
import pytest
from analogdb.client import Uploaded
from analogdb_generated.exceptions import ApiException
from dagster_aws.s3 import S3Resource
from PIL import Image
from scrape.models import (
    Caption,
    Keyword,
    PhotoMetadata,
    PostImages,
    RedditPost,
    S3Image,
)
from scrape.tagging import ImageTags, TaggingError

from .assets import (
    analogdb_permalinks,
    final_posts,
    patch_post_scores,
    post_images,
    post_tags,
    updated_reddit_comments,
    upload_posts,
)
from .convert import convert_create
from .resources import (
    AnalogDBResource,
    CamerasJsonResource,
    FilmsJsonResource,
    ImageProcessorResource,
    RedditResource,
    StorageResource,
    TaggerResource,
)
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


def make_post_images(count: int, image_count: int = 4) -> Result[PostImages]:
    data = {}
    for i in range(count):
        permalink = f"https://www.reddit.com/r/analog/comments/{i}/title"
        data[permalink] = PostImages(
            images=[
                S3Image(
                    resolution=str(r), url=f"https://s3/{i}/{r}.jpg", width=1, height=1
                )
                for r in range(image_count)
            ],
            colors=[],
            grayscale=False,
            width=1,
            height=1,
        )
    return Result(data=data, status={id: Status.SUCCESS for id in data})


def make_tags(word: str = "dog") -> ImageTags:
    return ImageTags(
        caption="A dog on a beach",
        tags=[word, "beach"],
        raw={"caption": "A dog on a beach", "tags": [word, "beach"]},
        model="m",
        version="v1",
    )


def no_tags(ids) -> Result[ImageTags]:
    return Result(data={}, status={id: Status.FAILED for id in ids})


def run_final_posts(post_images_result, tags_result):
    reddit_posts = make_reddit_posts(3)
    title_metadatas = Result(
        data={id: PhotoMetadata() for id in reddit_posts.data},
        status={id: Status.SUCCESS for id in reddit_posts.data},
    )
    context = dg.build_asset_context()
    return final_posts(
        context, reddit_posts, title_metadatas, post_images_result, tags_result
    )


class TestFinalPosts:
    def test_missing_tags_uploads_without_them(self):
        ids = list(make_reddit_posts(3).data)
        missing = ids[1]
        tags = Result(
            data={id: make_tags() for id in ids if id != missing},
            status={
                id: Status.FAILED if id == missing else Status.SUCCESS for id in ids
            },
        )

        result = run_final_posts(make_post_images(3), tags)

        assert result.successful_ids() == set(ids)
        assert result.data[missing].keywords == []
        assert result.data[missing].caption is None

    def test_tags_and_caption_in_create_body(self):
        ids = list(make_reddit_posts(3).data)
        tags = Result(
            data={id: make_tags() for id in ids},
            status={id: Status.SUCCESS for id in ids},
        )

        result = run_final_posts(make_post_images(3), tags)
        body = convert_create(result.data[ids[0]]).to_dict()

        assert result.data[ids[0]].keywords == [
            Keyword(word="dog", weight=1.0),
            Keyword(word="beach", weight=0.5),
        ]
        assert result.data[ids[0]].caption == Caption(
            caption="A dog on a beach",
            model="m",
            version="v1",
            raw={"caption": "A dog on a beach", "tags": ["dog", "beach"]},
        )
        assert body["keywords"] == [
            {"word": "dog", "weight": 1.0},
            {"word": "beach", "weight": 0.5},
        ]
        assert body["caption"] == {
            "caption": "A dog on a beach",
            "model": "m",
            "version": "v1",
            "raw": {"caption": "A dog on a beach", "tags": ["dog", "beach"]},
        }

    def test_too_few_images_fails(self):
        images = make_post_images(3)
        bad = next(iter(images.data))
        images.data[bad].images = images.data[bad].images[:3]

        result = run_final_posts(images, no_tags(images.data))

        assert result.status[bad] == Status.FAILED
        assert "4 images" in result.errors[bad]
        assert result.successful_ids() == set(images.data) - {bad}


class FakeTagger:
    def __init__(self, fail_titles=()):
        self.fail_titles = set(fail_titles)
        self.calls = []

    def tag_all(self, inputs, concurrency):
        self.calls.extend(inputs)
        return [
            TaggingError("refused") if i.title in self.fail_titles else make_tags()
            for i in inputs
        ]


def run_post_tags(reddit_posts, images, tagger):
    context = dg.build_asset_context()
    with patch.object(TaggerResource, "client", return_value=tagger):
        return post_tags(
            context,
            TaggerResource(),
            CamerasJsonResource(file_path="unused"),
            FilmsJsonResource(file_path="unused"),
            reddit_posts,
            images,
        )


def medium_images(count: int) -> Result[PostImages]:
    images = make_post_images(count)
    for p in images.data.values():
        p.images[1].resolution = "medium"
    return images


class TestPostTags:
    def test_tags_with_medium_image(self):
        reddit_posts = make_reddit_posts(2)
        tagger = FakeTagger()
        with patch.object(CamerasJsonResource, "client", return_value=[]), patch.object(
            FilmsJsonResource, "client", return_value=[]
        ):
            result = run_post_tags(reddit_posts, medium_images(2), tagger)

        assert result.successful_count() == 2
        assert [c.grayscale for c in tagger.calls] == [False, False]
        assert sorted(c.image_url for c in tagger.calls) == [
            "https://s3/0/1.jpg",
            "https://s3/1/1.jpg",
        ]

    def test_failure_still_uploads(self):
        reddit_posts = make_reddit_posts(3)
        bad = list(reddit_posts.data)[1]
        reddit_posts.data[bad].title = "refused"
        images = medium_images(3)
        images.data[list(images.data)[2]].images[1].resolution = "4"

        with patch.object(CamerasJsonResource, "client", return_value=[]), patch.object(
            FilmsJsonResource, "client", return_value=[]
        ):
            tags = run_post_tags(reddit_posts, images, FakeTagger({"refused"}))
        finals = run_final_posts(images, tags)

        assert tags.successful_count() == 1
        assert tags.status[bad] == Status.FAILED
        assert "refused" in tags.errors[bad]
        assert finals.successful_ids() == set(images.data)
        assert finals.data[bad].keywords == []


def run_with_client(fn, client: MagicMock):
    with patch.object(AnalogDBResource, "client", return_value=client):
        return fn()


class TestUploadPosts:
    def test_failure_tries_every_post(self):
        images = make_post_images(3)
        finals = run_final_posts(images, no_tags(images.data))
        bad = list(finals.data)[1]

        def upload(post):
            if post.permalink == bad:
                raise ApiException(status=500, reason="boom")
            return Uploaded.CREATED

        client = MagicMock()
        client.upload_post.side_effect = upload
        context = dg.build_asset_context()

        with pytest.raises(dg.Failure) as e:
            run_with_client(
                lambda: upload_posts(context, AnalogDBResource(), finals), client
            )

        assert client.upload_post.call_count == 3
        assert e.value.metadata["created"].value == 2
        assert e.value.metadata["failed"].value == 1

    def test_exists_counts_as_success(self):
        images = make_post_images(2)
        finals = run_final_posts(images, no_tags(images.data))
        client = MagicMock()
        client.upload_post.return_value = Uploaded.EXISTS
        context = dg.build_asset_context()

        result = run_with_client(
            lambda: upload_posts(context, AnalogDBResource(), finals), client
        )

        assert result.metadata == {"created": 0, "exists": 2, "failed": 0}


class TestPatchPosts:
    def test_failure_patches_rest(self):
        patches = [(i, adb.PostPatch(score=i)) for i in range(3)]
        client = MagicMock()
        client.patch_post.side_effect = [None, ApiException(status=500), None]
        context = dg.build_asset_context(partition_key="2024-01-01")

        with pytest.raises(dg.Failure):
            run_with_client(
                lambda: patch_post_scores(context, patches, AnalogDBResource()),
                client,
            )

        assert client.patch_post.call_count == 3


class TestAnalogdbPermalinks:
    def test_includes_removed_permalinks(self):
        client = MagicMock()
        client.get_latest_links.return_value = ["/r/analog/1", "/r/analog/2"]
        client.get_removed_links.return_value = ["/r/analog/2", "/r/analog/9"]
        context = dg.build_asset_context()
        with patch.object(AnalogDBResource, "client", return_value=client):
            links = analogdb_permalinks(context, AnalogDBResource())
        assert links == ["/r/analog/1", "/r/analog/2", "/r/analog/9"]
