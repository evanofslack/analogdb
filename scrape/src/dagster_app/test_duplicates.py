from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest

from . import duplicates
from .duplicates import (
    ReportDuplicatesConfig,
    report_duplicates,
)
from .resources import AnalogDBResource


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
        assert result.metadata["duplicates"] == 2
        assert result.metadata["reported"] == 1
        assert "reported" in result.metadata["pairs"].value

    def test_no_duplicates(self):
        client = MagicMock()
        client.report_duplicates.return_value = ([], 0)

        result = run_report(client, dry_run=False)

        assert result.metadata["pairs"].value == "None."
