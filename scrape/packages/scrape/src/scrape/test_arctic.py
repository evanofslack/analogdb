from unittest.mock import MagicMock

import pytest

from .arctic import PAGE_SIZE, ArcticShift, ArcticShiftError


def response(status: int, data=None, headers=None) -> MagicMock:
    resp = MagicMock()
    resp.status_code = status
    resp.headers = headers or {}
    resp.text = "error"
    resp.json.return_value = {"data": data, "error": None}
    return resp


def page(first: int, count: int) -> list[dict]:
    return [
        {"id": f"p{i}", "created_utc": 1000 + i} for i in range(first, first + count)
    ]


def make_client(responses: list[MagicMock]) -> tuple[ArcticShift, MagicMock, list]:
    session = MagicMock()
    session.headers = {}
    session.get.side_effect = responses
    sleeps: list[float] = []
    return ArcticShift(session=session, sleep=sleeps.append), session, sleeps


class TestPostIds:
    def test_pages_until_a_short_page(self):
        client, session, _ = make_client(
            [response(200, page(0, PAGE_SIZE)), response(200, page(PAGE_SIZE, 3))]
        )

        ids = client.post_ids("analog", 1000, 2000)

        assert len(ids) == PAGE_SIZE + 3
        first, second = [c.kwargs["params"] for c in session.get.call_args_list]
        assert first["after"] == 1000
        assert second["after"] == 1000 + PAGE_SIZE - 1
        assert second["before"] == 2000

    def test_dedupes_ids_repeated_across_pages(self):
        client, _, _ = make_client(
            [
                response(200, page(0, PAGE_SIZE)),
                response(200, page(PAGE_SIZE - 1, 2)),
            ]
        )

        ids = client.post_ids("analog", 1000, 2000)

        assert len(ids) == PAGE_SIZE + 1

    def test_retries_rate_limit_then_succeeds(self):
        client, _, sleeps = make_client(
            [
                response(429, headers={"x-ratelimit-reset": "7"}),
                response(422),
                response(200, page(0, 2)),
            ]
        )

        assert client.post_ids("analog", 1000, 2000) == ["p0", "p1"]
        assert sleeps == [8, 10]

    def test_gives_up_after_max_tries(self):
        client, session, _ = make_client([response(429)] * 5)

        with pytest.raises(ArcticShiftError):
            client.post_ids("analog", 1000, 2000)
        assert session.get.call_count == 5

    def test_raises_on_api_error(self):
        resp = response(200)
        resp.json.return_value = {"data": None, "error": "bad field"}
        client, _, _ = make_client([resp])

        with pytest.raises(ArcticShiftError, match="bad field"):
            client.post_ids("analog", 1000, 2000)
