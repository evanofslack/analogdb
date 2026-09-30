import json
from datetime import UTC, datetime
from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest
from dagster_aws.s3 import S3Resource
from scrape.models import MatchResult, PhotoMetadata, RedditComment

from .backfill import backfill_post_metadata, range_comments, rematch_post_metadata
from .resources import (
    AnalogDBResource,
    CamerasJsonResource,
    FilmsJsonResource,
    MetadataResource,
    RedditResource,
    StorageResource,
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
