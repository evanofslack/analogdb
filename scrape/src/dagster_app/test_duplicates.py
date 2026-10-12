from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest

from . import duplicates
from .duplicates import (
    BackfillImageHashesConfig,
    ReportDuplicatesConfig,
    backfill_image_hashes,
    report_duplicates,
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
        same_author=True,
        keep_url=f"https://x/low{keep}.jpg",
        duplicate_url=f"https://x/low{duplicate}.jpg",
    )


def run_report(client, **config):
    with patch.object(AnalogDBResource, "client", return_value=client):
        return report_duplicates(
            dg.build_asset_context(),
            config=ReportDuplicatesConfig(**config),
            analogdb=AnalogDBResource(),
        )


def report_client():
    client = MagicMock()
    dupes = [dupe(9156, 36395), dupe(9156, 40000)]
    client.get_duplicates.return_value = dupes
    client.report_duplicates.return_value = (dupes, 1)
    return client


class TestReportDuplicates:
    def test_dry_run_lists_without_reporting(self):
        client = report_client()

        result = run_report(client)

        client.report_duplicates.assert_not_called()
        client.delete_post.assert_not_called()
        assert result.metadata["dry_run"] is True
        assert result.metadata["duplicates"] == 2
        assert result.metadata["reported"] == 0
        assert result.metadata["duplicate_ids"] == "[36395, 40000]"
        table = result.metadata["pairs"].value
        assert "[36395](https://analogdb.com/post/36395)" in table
        assert "dry run" in table
        assert "| 0 | same |" in table

    def test_reports_without_deleting(self):
        client = report_client()

        result = run_report(client, dry_run=False)

        client.report_duplicates.assert_called_once_with()
        client.get_duplicates.assert_not_called()
        client.delete_post.assert_not_called()
        assert result.metadata["duplicates"] == 2
        assert result.metadata["reported"] == 1
        assert "reported" in result.metadata["pairs"].value

    def test_no_duplicates(self):
        client = MagicMock()
        client.report_duplicates.return_value = ([], 0)

        result = run_report(client, dry_run=False)

        assert result.metadata["pairs"].value == "None."
