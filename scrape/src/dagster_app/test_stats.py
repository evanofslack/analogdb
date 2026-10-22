from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest
from analogdb_generated.exceptions import ApiException

from .resources import AnalogDBResource
from .stats import rebuild_stats


def run_rebuild(client):
    with patch.object(AnalogDBResource, "client", return_value=client):
        return rebuild_stats(dg.build_asset_context(), analogdb=AnalogDBResource())


class TestRebuildStats:
    def test_records_the_build(self):
        client = MagicMock()
        client.rebuild_stats.return_value = adb.StatsBuild(
            id=12, build_ms=24366, films=38, themes=36
        )

        result = run_rebuild(client)

        client.rebuild_stats.assert_called_once_with()
        assert result.metadata == {
            "skipped": False,
            "snapshot_id": 12,
            "build_ms": 24366,
            "films": 38,
            "themes": 36,
        }

    def test_running_build_is_not_an_error(self):
        client = MagicMock()
        client.rebuild_stats.side_effect = ApiException(status=409)

        result = run_rebuild(client)

        assert result.metadata == {"skipped": True}

    def test_other_errors_fail(self):
        client = MagicMock()
        client.rebuild_stats.side_effect = ApiException(status=500)

        with pytest.raises(ApiException):
            run_rebuild(client)
