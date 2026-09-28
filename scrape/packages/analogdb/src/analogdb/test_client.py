import base64
import json

import pytest
from pytest_httpserver import HTTPServer
from werkzeug import Request, Response

from .client import Client
from .models import CameraCreate, FilmCreate, PostCreate, PostPatch, PostsFilter

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


def make_page(ids, next_page_id, next_page_url) -> dict:
    return {
        "posts": [make_post(i) for i in ids],
        "meta": {
            "total_posts": 6,
            "page_size": 2,
            "next_page_id": next_page_id,
            "next_page_url": next_page_url,
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
    def test_get_posts_all_follows_pages(self, client, httpserver: HTTPServer):
        httpserver.expect_ordered_request(
            "/v1/posts", query_string={"page_size": "6", "sort": "time"}
        ).respond_with_json(make_page([1, 2], 3, "/posts?page_id=3"))
        httpserver.expect_ordered_request(
            "/v1/posts",
            query_string={"page_size": "6", "sort": "time", "page_id": "3"},
        ).respond_with_json(make_page([3, 4], 5, "/posts?page_id=5"))
        httpserver.expect_ordered_request(
            "/v1/posts",
            query_string={"page_size": "6", "sort": "time", "page_id": "5"},
        ).respond_with_json(make_page([5], 0, ""))

        posts = client.get_posts_all(count=6)

        assert [p.id for p in posts] == [1, 2, 3, 4, 5]
        assert posts[0].images[0].resolution == "low"
        httpserver.check_assertions()

    def test_get_posts_with_filter(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/posts",
            query_string={
                "page_size": "20",
                "sort": "time",
                "time_start": "1",
                "time_end": "2",
            },
        ).respond_with_json(make_page([1], 0, ""))

        filter = PostsFilter(None, None, None, None, 1, 2)
        resp = client.get_posts(filter=filter)

        assert [p.id for p in resp.posts] == [1]

    def test_get_posts_null_posts(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/posts").respond_with_json(
            {"posts": None, "meta": {"next_page_url": ""}}
        )

        resp = client.get_posts()

        assert resp.posts == []
        assert client.get_posts_all(count=10) == []

    def test_get_latest_links(self, client, httpserver: HTTPServer):
        httpserver.expect_request("/v1/posts").respond_with_json(
            make_page([1, 2], 0, "")
        )

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

        resp = client.upload_film(film)

        assert resp.status_code == 201
        httpserver.check_assertions()

    def test_upload_camera_error_returns_status(self, client, httpserver: HTTPServer):
        httpserver.expect_request(
            "/v1/camera", method="POST", headers={"Authorization": AUTH}
        ).respond_with_json({"error": "bad"}, status=422)

        resp = client.upload_camera(CameraCreate(make="nikon", model="fm2"))

        assert resp.status_code == 422
        assert "bad" in resp.text

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

        resp = client.upload_post(post)

        assert resp.status_code == 201

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
