import base64
import json
import threading
import time

import pytest
from analogdb_generated.exceptions import ApiException
from pytest_httpserver import HTTPServer
from urllib3 import Retry
from urllib3.exceptions import HTTPError
from werkzeug import Request, Response

from .client import REQUEST_TIMEOUT, Client, Uploaded
from .models import (
    CameraCreate,
    FilmCreate,
    Keyword,
    PostCaption,
    PostCreate,
    PostExtraction,
    PostPatch,
    PostsFilter,
)

AUTH = "Basic " + base64.b64encode(b"user:pass").decode()


def make_post(id: int) -> dict:
    return {
        "id": id,
        "title": f"post {id}",
        "author": "u/author",
        "permalink": f"/r/analog/{id}",
        "score": 1,
        "timestamp": 1700000000 + id,
        "nsfw": False,
        "grayscale": False,
        "sprocket": False,
        "images": [
            {"url": "https://x/1.jpg", "resolution": "low", "width": 1, "height": 1}
        ],
    }


def make_page(ids, next_cursor) -> dict:
    return {
        "posts": [make_post(i) for i in ids],
        "meta": {
            "total_posts": 6,
            "page_size": 2,
            "next_cursor": next_cursor,
        },
    }


@pytest.fixture
def client(httpserver: HTTPServer):
    return Client(
        base_url=httpserver.url_for("").rstrip("/"),
        username="user",
        password="pass",
    )


class TestFilterToParams:
    def test_filter_none_returns_defaults(self, client):
        assert client._filter_to_params(None) == {"page_size": 20, "sort": "time"}

    def test_filter_with_partial_params(self, client):
        filter_obj = PostsFilter(
            count=None,
            nsfw=False,
            grayscale=None,
            sprocket=True,
            time_start=10,
            time_end=None,
        )
        params = client._filter_to_params(filter_obj)
        assert params == {
            "page_size": 20,
            "sort": "time",
            "nsfw": False,
            "sprocket": True,
            "time_start": 10,
        }


class TestGetPosts:
    def test_get_posts_all_follows_cursors(self, client, httpserver: HTTPServer):
        httpserver.expect_ordered_request(
            "/v1/posts", query_string={"page_size": "6", "sort": "time"}
        ).respond_with_json(make_page([1, 2], "c1"))
        httpserver.expect_ordered_request(
            "/v1/posts",
            query_string={"page_size": "6", "sort": "time", "cursor": "c1"},
        ).respond_with_json(make_page([3, 4], "c2"))
        httpserver.expect_ordered_request(
            "/v1/posts",
            query_string={"page_size": "6", "sort": "time", "cursor": "c2"},
        ).respond_with_json(make_page([5], ""))

        posts = client.get_posts_all(count=6)

        assert [p.id for p in posts] == [1, 2, 3, 4, 5]
        assert posts[0].images[0].resolution == "low"
        assert all("page_id" not in r.args for r, _ in httpserver.log)
        httpserver.check_assertions()

    def test_get_posts_all_stops_at_count(self, client, httpserver: HTTPServer):
        httpserver.expect_ordered_request("/v1/posts").respond_with_json(
            make_page([1, 2], "c1")
        )
        httpserver.expect_ordered_request(
            "/v1/posts", query_string={"page_size": "3", "sort": "time", "cursor": "c1"}
        ).respond_with_json(make_page([3, 4], "c2"))

        posts = client.get_posts_all(count=3)

        assert [p.id for p in posts] == [1, 2, 3]
        assert len(httpserver.log) == 2

    def test_get_posts_with_filter(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/posts",
            query_string={
                "page_size": "20",
                "sort": "time",
                "time_start": "1",
                "time_end": "2",
            },
        ).respond_with_json(make_page([1], ""))

        filter = PostsFilter(None, None, None, None, 1, 2)
        resp = client.get_posts(filter=filter)

        assert [p.id for p in resp.posts] == [1]

    def test_get_posts_null_posts(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/posts").respond_with_json(
            {"posts": None, "meta": {"next_cursor": ""}}
        )

        resp = client.get_posts()

        assert resp.posts == []
        assert client.get_posts_all(count=10) == []

    def test_get_latest_links(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/posts").respond_with_json(make_page([1, 2], ""))

        assert client.get_latest_links(2) == ["/r/analog/1", "/r/analog/2"]


class TestRequests:
    def test_reads_have_user_agent_and_no_auth(self, client, httpserver: HTTPServer):
        seen = {}

        def handler(request: Request):
            seen["ua"] = request.headers.get("User-Agent")
            seen["auth"] = request.headers.get("Authorization")
            return Response(
                json.dumps({"films": [{"id": 1, "make": "kodak"}]}),
                mimetype="application/json",
            )

        httpserver.expect_request("/v1/films", method="GET").respond_with_handler(
            handler
        )

        films = client.get_films()

        assert films[0].make == "kodak"
        assert seen["ua"].startswith("analogdb-scraper/")
        assert seen["auth"] is None

    def test_get_cameras(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/cameras", method="GET").respond_with_json(
            {"cameras": [{"id": 2, "make": "nikon", "model": "fm2"}]}
        )

        cameras = client.get_cameras()

        assert cameras[0].id == 2
        assert cameras[0].model == "fm2"

    def test_upload_film_posts_to_film(self, client, httpserver: HTTPServer):
        film = FilmCreate(
            type="portra 400",
            make="kodak",
            speed=400,
            color_type="color",
            description="d",
        )
        httpserver.expect_request(
            "/v1/film",
            method="POST",
            headers={"Authorization": AUTH},
            json=film.to_dict(),
        ).respond_with_json({"message": "ok", "film": film.to_dict()}, status=201)

        client.upload_film(film)

        httpserver.check_assertions()

    def test_upload_camera_error_raises(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/camera", method="POST", headers={"Authorization": AUTH}
        ).respond_with_json({"error": "bad"}, status=422)

        with pytest.raises(ApiException) as e:
            client.upload_camera(CameraCreate(make="nikon", model="fm2"))

        assert e.value.status == 422
        assert "bad" in e.value.body
        assert len(httpserver.log) == 1

    def test_upload_post(self, client, httpserver: HTTPServer):
        post = PostCreate(title="t", permalink="/r/analog/1", timestamp=1, images=[])
        httpserver.expect_request(
            "/v1/post",
            method="POST",
            headers={"Authorization": AUTH},
            json={
                "title": "t",
                "permalink": "/r/analog/1",
                "timestamp": 1,
                "images": [],
            },
        ).respond_with_json({"message": "ok"}, status=201)

        assert client.upload_post(post) == Uploaded.CREATED

    def test_patch_post_sends_only_set_fields(self, client, httpserver: HTTPServer):
        seen = {}

        def handler(request: Request):
            seen["body"] = request.get_data(as_text=True)
            seen["ua"] = request.headers.get("User-Agent")
            return Response(json.dumps({"message": "ok"}), mimetype="application/json")

        httpserver.expect_request(
            "/v1/post/7", method="PATCH", headers={"Authorization": AUTH}
        ).respond_with_handler(handler)

        client.patch_post(7, PostPatch(score=5))

        assert json.loads(seen["body"]) == {"score": 5}
        assert seen["ua"].startswith("analogdb-scraper/")

    def test_patch_post_empty_is_skipped(self, client, httpserver: HTTPServer):
        assert client.patch_post(7, PostPatch()) is None
        assert len(httpserver.log) == 0


def post_create() -> PostCreate:
    return PostCreate(title="t", permalink="/r/analog/1", timestamp=1, images=[])


class TestUploadPostStatus:
    def test_conflict_is_exists(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/post", method="POST").respond_with_json(
            {"error": "exists"}, status=409
        )

        assert client.upload_post(post_create()) == Uploaded.EXISTS
        assert len(httpserver.log) == 1

    def test_server_error_raises_without_retry(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/post", method="POST").respond_with_json(
            {"error": "boom"}, status=500
        )

        with pytest.raises(ApiException) as e:
            client.upload_post(post_create())

        assert e.value.status == 500
        assert len(httpserver.log) == 1

    def test_unavailable_is_retried(self, client, httpserver: HTTPServer):
        httpserver.expect_ordered_request("/v1/post", method="POST").respond_with_json(
            {"error": "down"}, status=503
        )
        httpserver.expect_ordered_request("/v1/post", method="POST").respond_with_json(
            {"message": "ok"}, status=201
        )

        assert client.upload_post(post_create()) == Uploaded.CREATED
        assert len(httpserver.log) == 2

    def test_unauthorized_is_not_retried(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/post", method="POST").respond_with_json(
            {"error": "unauthorized"}, status=401
        )

        with pytest.raises(ApiException) as e:
            client.upload_post(post_create())

        assert e.value.status == 401
        assert len(httpserver.log) == 1


class TestTimeout:
    def test_default_timeout(self, client):
        assert client.timeout == REQUEST_TIMEOUT == (10.0, 30.0)

    def test_hanging_server_times_out(self, httpserver: HTTPServer):
        release = threading.Event()

        def handler(request: Request):
            release.wait(5)
            return Response(json.dumps({"films": []}), mimetype="application/json")

        httpserver.expect_request("/v1/films").respond_with_handler(handler)
        client = Client(
            base_url=httpserver.url_for("").rstrip("/"),
            timeout=(1.0, 0.2),
            retries=Retry(0),
        )

        start = time.monotonic()
        try:
            with pytest.raises(HTTPError):
                client.get_films()
        finally:
            release.set()

        assert time.monotonic() - start < 2


def extraction(post_id: int) -> PostExtraction:
    return PostExtraction(
        post_id=post_id,
        extractor_version="v",
        model="m",
        input="title: x",
        input_hash="h",
        raw={"cameras": [], "films": [], "lenses": []},
        unmatched=[{"kind": "camera", "raw": "Nikon FM", "key": "nikonfm"}],
    )


class TestExtractions:
    def test_upsert_chunks_and_sums(self, client, httpserver: HTTPServer):
        bodies = []

        def handler(request: Request):
            body = json.loads(request.get_data(as_text=True))
            bodies.append(body)
            ids = [e["post_id"] for e in body["extractions"]]
            resp = {"written": len(ids) - 1, "skipped": ids[-1:]}
            return Response(json.dumps(resp), mimetype="application/json")

        httpserver.expect_request(
            "/v1/admin/extractions", method="POST", headers={"Authorization": AUTH}
        ).respond_with_handler(handler)

        written, skipped = client.upsert_extractions(
            [extraction(i) for i in range(1, 6)], chunk_size=2
        )

        assert [len(b["extractions"]) for b in bodies] == [2, 2, 1]
        assert bodies[0]["extractions"][0]["raw"] == {
            "cameras": [],
            "films": [],
            "lenses": [],
        }
        assert written == 2
        assert skipped == [2, 4, 5]

    def test_iter_follows_before_id(self, client, httpserver: HTTPServer):
        def page(ids, next_before):
            return {
                "extractions": [extraction(i).to_dict() for i in ids],
                "next_before_id": next_before,
            }

        httpserver.expect_ordered_request(
            "/v1/admin/extractions",
            query_string={"has_unmatched": "true", "full": "true", "limit": "500"},
        ).respond_with_json(page([9, 8], 8))
        httpserver.expect_ordered_request(
            "/v1/admin/extractions",
            query_string={
                "has_unmatched": "true",
                "full": "true",
                "before_id": "8",
                "limit": "500",
            },
        ).respond_with_json(page([7], None))

        got = [
            e.post_id for e in client.iter_extractions(has_unmatched=True, full=True)
        ]
        assert got == [9, 8, 7]


class TestScrapeRoutes:
    def test_get_post_ids(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/ids", method="GET").respond_with_json(
            {"ids": [1, 2, 3]}
        )

        assert client.get_post_ids() == [1, 2, 3]

    def test_encode_posts_returns_failed(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/encode",
            method="PUT",
            headers={"Authorization": AUTH},
            json={"ids": [1, 2, 3], "batch_size": 20},
        ).respond_with_json({"message": "ok", "failed_ids": [2]})

        assert client.encode_posts([1, 2, 3], batch_size=20) == [2]

    def test_encode_no_ids_is_skipped(self, client, httpserver: HTTPServer):
        assert client.encode_posts([]) == []
        assert len(httpserver.log) == 0

    def test_get_missing_captions(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/scrape/captions/missing",
            method="GET",
            query_string={"version": "v1"},
            headers={"Authorization": AUTH},
        ).respond_with_json({"ids": [4, 5]})

        assert client.get_missing_captions("v1") == [4, 5]

    def test_get_missing_captions_null(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/scrape/captions/missing", query_string=""
        ).respond_with_json({"ids": None})

        assert client.get_missing_captions() == []

    def test_get_missing_vectors(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/scrape/vectors/missing",
            method="GET",
            headers={"Authorization": AUTH},
        ).respond_with_json({"ids": [4, 9], "extra": 1})

        assert client.get_missing_vectors() == [4, 9]

    def test_get_missing_vectors_null(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/scrape/vectors/missing").respond_with_json(
            {"ids": None, "extra": 0}
        )

        assert client.get_missing_vectors() == []

    def test_patch_post_caption(self, client, httpserver: HTTPServer):
        seen = {}

        def handler(request: Request):
            seen["body"] = json.loads(request.get_data(as_text=True))
            return Response(json.dumps({"message": "ok"}), mimetype="application/json")

        httpserver.expect_request("/v1/post/7", method="PATCH").respond_with_handler(
            handler
        )

        client.patch_post(
            7,
            PostPatch(
                caption=PostCaption(
                    caption=None, model="m", version="v1-text", raw={"tags": ["dog"]}
                ),
                keywords=[Keyword(word="dog", weight=1.0)],
            ),
        )

        assert seen["body"] == {
            "caption": {"model": "m", "version": "v1-text", "raw": {"tags": ["dog"]}},
            "keywords": [{"word": "dog", "weight": 1.0}],
        }
