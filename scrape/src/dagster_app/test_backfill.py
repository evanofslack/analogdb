import json
from datetime import UTC, datetime
from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest
from dagster_aws.s3 import S3Resource
from scrape.models import MatchResult, PhotoMetadata, RedditComment
from scrape.tagging import ImageTags, TaggingError

from . import backfill
from .backfill import (
    BackfillCaptionsConfig,
    ReencodeVectorsConfig,
    backfill_post_captions,
    backfill_post_metadata,
    range_comments,
    reencode_post_vectors,
    rematch_post_metadata,
)
from .resources import (
    AnalogDBResource,
    CamerasJsonResource,
    FilmsJsonResource,
    MetadataResource,
    RedditResource,
    StorageResource,
    TaggerResource,
)
from .test_metadata_flow import CAMERAS_JSON, LIVE_CAMERAS, post


def comment_json(author):
    return {"body": "hi", "score": 1, "author": author, "time": 1, "permalink": "p"}


class FakeStorage:
    def __init__(self, cached):
        self.cached = cached
        self.put = {}

    def get_object(self, bucket, key):
        return self.cached.get(key)

    def put_object(self, bucket, key, body, content_type):
        self.put[key] = body


class TestRangeComments:
    def run(self, storage, scraper):
        with patch.object(RedditResource, "client", return_value=scraper):
            posts = [post(i) for i in range(1, 7)]
            return range_comments(
                dg.build_asset_context(), posts, storage, RedditResource()
            )

    def test_cache_hit_miss_and_write_back(self):
        storage = FakeStorage({"1.json": json.dumps([comment_json("u/me")]).encode()})
        scraper = MagicMock()
        scraper.scrape_comments.side_effect = lambda permalink: (
            [RedditComment(body="new", score=1, author="u/me", time=2, permalink="p")]
            if permalink.endswith("/2")
            else []
        )
        comments = self.run(storage, scraper)
        assert comments[1][0].author == "u/me"
        assert comments[2][0].body == "new"
        assert json.loads(storage.put["2.json"])[0]["body"] == "new"
        assert scraper.scrape_comments.call_count == 5

    def test_all_reddit_failures_raise(self):
        scraper = MagicMock()
        scraper.scrape_comments.side_effect = RuntimeError("down")
        with pytest.raises(dg.Failure):
            self.run(FakeStorage({}), scraper)


def resources():
    return dict(
        cameras_json=CamerasJsonResource(file_path="unused"),
        films_json=FilmsJsonResource(file_path="unused"),
    )


class TestBackfillPostMetadata:
    def test_range_runs_as_one_window_in_chunks(self):
        client = MagicMock()
        client.get_posts_all.return_value = [post(2, timestamp=5), post(1, timestamp=1)]
        client.get_cameras.return_value = LIVE_CAMERAS
        client.get_films.return_value = []
        client.upsert_extractions.return_value = (2, [])
        extractor = MagicMock()
        extractor.extract.side_effect = lambda inputs, *a: [
            MatchResult(
                proposed=PhotoMetadata(camera_make="nikon", camera_model="f3"),
                raw={"cameras": [{"text": "Nikon F3"}], "films": [], "lenses": []},
            )
            for _ in inputs
        ]
        storage = FakeStorage({"1.json": b"[]", "2.json": b"[]"})
        context = dg.build_asset_context(
            partition_key_range=dg.PartitionKeyRange("2024-01-01", "2024-01-31")
        )
        with (
            patch.object(AnalogDBResource, "client", return_value=client),
            patch.object(MetadataResource, "client", return_value=extractor),
            patch.object(CamerasJsonResource, "client", return_value=CAMERAS_JSON),
            patch.object(FilmsJsonResource, "client", return_value=[]),
            patch.object(StorageResource, "get_object", side_effect=storage.get_object),
        ):
            result = backfill_post_metadata(
                context,
                analogdb=AnalogDBResource(),
                metadata=MetadataResource(openai_model="model-x"),
                reddit=RedditResource(),
                storage=StorageResource(s3_resource=S3Resource()),
                **resources(),
            )

        filter = client.get_posts_all.call_args.kwargs["filter"]
        assert filter.time_start == int(datetime(2024, 1, 1, tzinfo=UTC).timestamp())
        assert filter.time_end == int(datetime(2024, 2, 1, tzinfo=UTC).timestamp())
        assert [c.args[0] for c in client.patch_post.call_args_list] == [1, 2]
        stored = client.upsert_extractions.call_args.args[0]
        assert {r.model for r in stored} == {"model-x"}
        assert result.metadata["patched"] == 2
        assert result.metadata["camera_patches"] == 2


class TestRematch:
    def test_new_alias_changes_only_affected_posts(self):
        client = MagicMock()
        client.get_cameras.return_value = LIVE_CAMERAS
        client.get_films.return_value = []
        client.get_posts_all.return_value = [
            post(1),
            post(2, camera_make="nikon", camera_model="f3"),
        ]
        client.upsert_extractions.return_value = (1, [])
        client.iter_extractions.return_value = [
            adb.PostExtraction(
                post_id=1,
                extractor_version="old",
                model="m",
                input="title: Nikon N2020 at dusk",
                input_hash="h",
                raw={
                    "cameras": [
                        {"text": "Nikon N2020", "make": "Nikon", "model": "N2020"}
                    ]
                },
                unmatched=[
                    {"kind": "camera", "raw": "Nikon N2020", "key": "nikonn2020"}
                ],
            ),
            adb.PostExtraction(
                post_id=2,
                extractor_version="old",
                model="m",
                input="title: Nikon F3",
                input_hash="h",
                raw={"cameras": [{"text": "Nikon F3", "make": "Nikon", "model": "F3"}]},
                unmatched=[],
            ),
            adb.PostExtraction(
                post_id=99,
                extractor_version="old",
                model="m",
                input="x",
                input_hash="h",
                raw={},
            ),
        ]
        with (
            patch.object(AnalogDBResource, "client", return_value=client),
            patch.object(CamerasJsonResource, "client", return_value=CAMERAS_JSON),
            patch.object(FilmsJsonResource, "client", return_value=[]),
            patch.object(
                MetadataResource, "client", side_effect=AssertionError("no LLM")
            ),
        ):
            result = rematch_post_metadata(
                dg.build_asset_context(), analogdb=AnalogDBResource(), **resources()
            )

        client.patch_post.assert_called_once()
        post_id, patch_body = client.patch_post.call_args.args
        assert post_id == 1
        assert patch_body.to_dict() == {"camera_make": "nikon", "camera_model": "f501"}
        stored = client.upsert_extractions.call_args.args[0]
        assert [(r.post_id, r.unmatched, r.extractor_version) for r in stored] == [
            (1, [], "old")
        ]
        assert result.metadata["posts"] == 2
        assert result.metadata["unchanged"] == 1


class FakeCaptionClient:
    def __init__(self, posts):
        self.posts = posts
        self.versions = {}
        self.patches = []

    def get_missing_captions(self, version=None):
        return [p.id for p in self.posts if self.versions.get(p.id) != version]

    def get_posts_all(self, count, filter=None):
        return self.posts

    def patch_post(self, id, patch):
        self.versions[id] = patch.caption.version
        self.patches.append((id, patch))


class FakeTagger:
    def __init__(self):
        self.chunks = []

    def tag_all(self, inputs, concurrency):
        self.grayscale = [i.grayscale for i in inputs]
        self.chunks.append([i.title for i in inputs])
        return [self.tag(i.title) for i in inputs]

    def tag(self, title):
        if title == "fail":
            return TaggingError("refused")
        text_only = title == "text"
        return ImageTags(
            caption=None if text_only else "A dog",
            tags=["dog", "beach"],
            raw={"tags": ["dog", "beach"]},
            model="m",
            version="v1-text" if text_only else "v1",
        )


def medium_post(id, title="t"):
    images = [adb.Image(url=f"https://cdn/{id}.jpg", resolution="medium")]
    return post(id, title=title, images=images)


class TestBackfillCaptions:
    def run(self, client, tagger, **config):
        with (
            patch.object(AnalogDBResource, "client", return_value=client),
            patch.object(TaggerResource, "client", return_value=tagger),
            patch.object(CamerasJsonResource, "client", return_value=[]),
            patch.object(FilmsJsonResource, "client", return_value=[]),
            patch.object(backfill, "BACKFILL_CHUNK", 2),
        ):
            return backfill_post_captions(
                dg.build_asset_context(),
                config=BackfillCaptionsConfig(**config),
                analogdb=AnalogDBResource(),
                tagger=TaggerResource(),
                **resources(),
            )

    def posts(self):
        titles = ["a", "b", "text", "fail", "c"]
        return [medium_post(i, t) for i, t in enumerate(titles, start=1)]

    def test_chunks_patches_and_counts(self):
        client = FakeCaptionClient(self.posts())
        tagger = FakeTagger()

        result = self.run(client, tagger)

        assert tagger.chunks == [["a", "b"], ["text", "fail"], ["c"]]
        assert result.metadata["written"] == 4
        assert result.metadata["text_only"] == 1
        assert result.metadata["tag_failed"] == 1
        id, first = client.patches[0]
        assert id == 1
        assert first.to_dict() == {
            "caption": {
                "caption": "A dog",
                "model": "m",
                "version": "v1",
                "raw": {"tags": ["dog", "beach"]},
            },
            "keywords": [
                {"word": "dog", "weight": 1.0},
                {"word": "beach", "weight": 0.5},
            ],
        }

    def test_rerun_picks_up_where_it_stopped(self):
        client = FakeCaptionClient(self.posts())
        self.run(client, FakeTagger())
        tagger = FakeTagger()

        result = self.run(client, tagger)

        assert tagger.chunks == [["fail"]]
        assert result.metadata["written"] == 0

    def test_retry_text_only(self):
        client = FakeCaptionClient(self.posts())
        self.run(client, FakeTagger())
        tagger = FakeTagger()

        self.run(client, tagger, retry_text_only=True)

        assert tagger.chunks == [["text", "fail"]]

    def test_passes_grayscale(self):
        posts = [medium_post(1), medium_post(2)]
        posts[1].grayscale = True
        tagger = FakeTagger()
        self.run(FakeCaptionClient(posts), tagger)
        assert tagger.grayscale == [False, True]

    def test_limit(self):
        tagger = FakeTagger()
        self.run(FakeCaptionClient(self.posts()), tagger, limit=1)
        assert tagger.chunks == [["a"]]

    def test_patch_failure_raises(self):
        client = FakeCaptionClient(self.posts()[:1])
        client.patch_post = MagicMock(side_effect=RuntimeError("down"))
        with pytest.raises(dg.Failure):
            self.run(client, FakeTagger())


class TestReencodeVectors:
    def run(self, client, **config):
        with (
            patch.object(AnalogDBResource, "client", return_value=client),
            dg.build_asset_context() as context,
        ):
            return reencode_post_vectors(
                context,
                config=ReencodeVectorsConfig(**config),
                analogdb=AnalogDBResource(),
            )

    def test_failed_ids_retried_one_at_a_time(self):
        client = MagicMock()
        client.get_missing_vectors.return_value = [1, 2, 3, 4, 5]

        def encode(ids, batch_size):
            if ids == [3, 4]:
                raise RuntimeError("timeout")
            if ids == [1, 2]:
                return [2]
            return [4] if ids == [4] else []

        client.encode_posts.side_effect = encode

        with pytest.raises(dg.Failure) as e:
            self.run(client, batch_size=2)

        calls = [(c.args[0], c.args[1]) for c in client.encode_posts.call_args_list]
        assert calls == [
            ([1, 2], 2),
            ([3, 4], 2),
            ([5], 2),
            ([2], 1),
            ([3], 1),
            ([4], 1),
        ]
        assert e.value.metadata["retried"].value == 3
        assert e.value.metadata["failed"].value == 1

    def test_config_ids_and_limit(self):
        client = MagicMock()
        client.encode_posts.return_value = []

        result = self.run(client, ids=[7, 8, 9], limit=2)

        client.get_post_ids.assert_not_called()
        assert client.encode_posts.call_args.args == ([7, 8], 20)
        assert result.metadata == {"posts": 2, "retried": 0, "failed": 0}

    def test_id_source_order(self):
        client = MagicMock()
        client.get_missing_vectors.return_value = [3, 4]
        client.get_post_ids.return_value = [1, 2, 3, 4]
        client.encode_posts.return_value = []

        self.run(client, ids=[9])
        assert client.encode_posts.call_args.args == ([9], 20)
        client.get_missing_vectors.assert_not_called()

        self.run(client)
        assert client.encode_posts.call_args.args == ([3, 4], 20)
        client.get_post_ids.assert_not_called()

        result = self.run(client, missing_only=False, limit=3)
        assert client.encode_posts.call_args.args == ([1, 2, 3], 20)
        assert result.metadata == {"posts": 3, "retried": 0, "failed": 0}

    def test_nothing_missing_is_noop(self):
        client = MagicMock()
        client.get_missing_vectors.return_value = []

        result = self.run(client)

        client.encode_posts.assert_not_called()
        assert result.metadata == {"posts": 0}

    def test_fail_on_error_off_does_not_raise(self):
        client = MagicMock()
        client.get_missing_vectors.return_value = [1, 2]
        client.encode_posts.side_effect = lambda ids, size: list(ids)

        result = self.run(client, fail_on_error=False)

        assert result.metadata["failed"] == 2
        assert result.metadata["failed_ids"] == "[1, 2]"
