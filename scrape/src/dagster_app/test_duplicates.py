from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest
from analogdb.client import Deleted

from . import duplicates
from .duplicates import (
    BackfillImageHashesConfig,
    PruneDuplicatesConfig,
    backfill_image_hashes,
    prune_duplicates,
)
from .resources import AnalogDBResource


def missing(n: int):
    return [
        adb.ImageHashMissing(id=i, url=f"https://x/low{i}.jpg") for i in range(1, n + 1)
    ]


def run_backfill(client, config=None, download=None):
    download = download or (lambda url: "00000000000000ff")
    with (
        patch.object(AnalogDBResource, "client", return_value=client),
        patch.object(duplicates, "download_hash", side_effect=download),
        patch.object(duplicates, "BACKFILL_CHUNK", 2),
    ):
        return backfill_image_hashes(
            dg.build_asset_context(),
            config=config or BackfillImageHashesConfig(),
            analogdb=AnalogDBResource(),
        )


class TestBackfillImageHashes:
    def test_writes_hashes_in_chunks(self):
        client = MagicMock()
        client.get_missing_hashes.return_value = missing(5)
        client.set_image_hashes.side_effect = lambda hashes: len(hashes)

        result = run_backfill(client)

        written = [
            [h.id for h in c.args[0]] for c in client.set_image_hashes.call_args_list
        ]
        assert written == [[1, 2], [3, 4], [5]]
        assert result.metadata["written"] == 5
        assert result.metadata["download_failed"] == 0

    def test_failed_download_is_skipped(self):
        client = MagicMock()
        client.get_missing_hashes.return_value = missing(3)
        client.set_image_hashes.side_effect = lambda hashes: len(hashes)

        def download(url):
            if url.endswith("low2.jpg"):
                raise OSError("gone")
            return "00000000000000ff"

        result = run_backfill(client, download=download)

        ids = [h.id for c in client.set_image_hashes.call_args_list for h in c.args[0]]
        assert ids == [1, 3]
        assert result.metadata["written"] == 2
        assert result.metadata["download_failed"] == 1

    def test_limit(self):
        client = MagicMock()
        client.get_missing_hashes.return_value = missing(5)
        client.set_image_hashes.side_effect = lambda hashes: len(hashes)

        result = run_backfill(client, BackfillImageHashesConfig(limit=3))

        assert result.metadata["posts"] == 3

    def test_failed_write_fails_after_every_chunk(self):
        client = MagicMock()
        client.get_missing_hashes.return_value = missing(4)
        client.set_image_hashes.side_effect = [RuntimeError("down"), 2]

        with pytest.raises(dg.Failure) as e:
            run_backfill(client)

        assert client.set_image_hashes.call_count == 2
        assert e.value.metadata["write_failed"].value == 2
        assert e.value.metadata["written"].value == 2


def dupe(keep: int, duplicate: int) -> adb.DuplicatePost:
    return adb.DuplicatePost(
        keep_id=keep,
        duplicate_id=duplicate,
        distance=0,
        keep_url=f"https://x/low{keep}.jpg",
        duplicate_url=f"https://x/low{duplicate}.jpg",
    )


def run_prune(client, **config):
    with patch.object(AnalogDBResource, "client", return_value=client):
        return prune_duplicates(
            dg.build_asset_context(),
            config=PruneDuplicatesConfig(**config),
            analogdb=AnalogDBResource(),
        )


def prune_client():
    client = MagicMock()
    client.get_duplicates.return_value = [dupe(9156, 36395), dupe(9156, 40000)]
    client.delete_post.return_value = Deleted.DELETED
    return client


class TestPruneDuplicates:
    def test_dry_run_lists_without_deleting(self):
        client = prune_client()

        result = run_prune(client)

        client.delete_post.assert_not_called()
        assert result.metadata["dry_run"] is True
        assert result.metadata["duplicates"] == 2
        assert result.metadata["duplicate_ids"] == "[36395, 40000]"
        table = result.metadata["pairs"].value
        assert "[36395](https://analogdb.com/post/36395)" in table
        assert "dry run" in table

    def test_deletes_with_reason(self):
        client = prune_client()
        client.delete_post.side_effect = [Deleted.DELETED, Deleted.MISSING]

        result = run_prune(client, dry_run=False)

        assert [c.args for c in client.delete_post.call_args_list] == [
            (36395,),
            (40000,),
        ]
        assert {c.kwargs["reason"] for c in client.delete_post.call_args_list} == {
            "duplicate of 9156"
        }
        assert result.metadata["deleted"] == 1
        assert result.metadata["missing"] == 1

    def test_max_deletes_fails_before_deleting(self):
        client = prune_client()

        with pytest.raises(dg.Failure):
            run_prune(client, dry_run=False, max_deletes=1)

        client.delete_post.assert_not_called()

    def test_max_deletes_ignored_on_dry_run(self):
        result = run_prune(prune_client(), max_deletes=1)

        assert result.metadata["duplicates"] == 2

    def test_failed_delete_fails_after_trying_all(self):
        client = prune_client()
        client.delete_post.side_effect = [RuntimeError("down"), Deleted.DELETED]

        with pytest.raises(dg.Failure) as e:
            run_prune(client, dry_run=False)

        assert client.delete_post.call_count == 2
        assert e.value.metadata["failed"].value == 1
        assert e.value.metadata["deleted"].value == 1

    def test_no_duplicates(self):
        client = MagicMock()
        client.get_duplicates.return_value = []

        result = run_prune(client, dry_run=False)

        client.delete_post.assert_not_called()
        assert result.metadata["pairs"].value == "None."
