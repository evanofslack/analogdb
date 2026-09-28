import functools
import time
from dataclasses import dataclass
from importlib.metadata import version
from typing import Any, Dict, List, Optional

from analogdb_generated import ApiClient, ApiResponse, Configuration
from analogdb_generated.api.camera_api import CameraApi
from analogdb_generated.api.cameras_api import CamerasApi
from analogdb_generated.api.film_api import FilmApi
from analogdb_generated.api.films_api import FilmsApi
from analogdb_generated.api.post_api import PostApi
from analogdb_generated.api.posts_api import PostsApi
from analogdb_generated.exceptions import ApiException
from analogdb_generated.models.server_post_response import ServerPostResponse

from .models import (
    Camera,
    CameraCreate,
    Film,
    FilmCreate,
    Post,
    PostCreate,
    PostPatch,
    PostsFilter,
)

DEFAULT_PAGE_SIZE = 20
DEFAULT_SORT = "time"
USER_AGENT = f"analogdb-scraper/{version('analogdb')}"


def retry(delay=1, times=5):
    def outer_wrapper(function):
        @functools.wraps(function)
        def inner_wrapper(*args, **kwargs):
            final_excep = None
            for counter in range(times):
                if counter > 0:
                    time.sleep(delay)
                final_excep = None
                try:
                    return function(*args, **kwargs)
                except Exception as e:
                    final_excep = e
            if final_excep is not None:
                raise final_excep

        return inner_wrapper

    return outer_wrapper


@dataclass
class Response:
    status_code: int
    text: str


class Client:
    def __init__(
        self,
        base_url: str = "https://api.analogdb.com",
        username: Optional[str] = None,
        password: Optional[str] = None,
    ):
        self.base_url = base_url

        config = Configuration(host=f"{base_url}/v1")
        if username and password:
            config.username = username
            config.password = password
        self.api_client = ApiClient(config)
        self.api_client.user_agent = USER_AGENT

        self.posts_api = PostsApi(self.api_client)
        self.post_api = PostApi(self.api_client)
        self.films_api = FilmsApi(self.api_client)
        self.film_api = FilmApi(self.api_client)
        self.cameras_api = CamerasApi(self.api_client)
        self.camera_api = CameraApi(self.api_client)

    def get_posts(
        self,
        count: int = DEFAULT_PAGE_SIZE,
        filter: Optional[PostsFilter] = None,
        page_id: Optional[int] = None,
    ) -> ServerPostResponse:
        params = self._filter_to_params(filter)
        params["page_size"] = count
        if page_id is not None:
            params["page_id"] = page_id

        resp = self.posts_api.posts_get(**params)
        if resp.posts is None:
            resp.posts = []
        return resp

    def get_posts_all(
        self, count: int = 20, filter: Optional[PostsFilter] = None
    ) -> List[Post]:
        analog_posts = []
        page_id = None

        num = DEFAULT_PAGE_SIZE * 5
        if num > count:
            num = count
        while len(analog_posts) < count:
            resp = self.get_posts(num, filter, page_id)
            for p in resp.posts:
                if len(analog_posts) >= count:
                    break
                analog_posts.append(p)

            if not resp.meta:
                break
            # no more pages
            if not resp.meta.next_page_url:
                break
            page_id = resp.meta.next_page_id

        return analog_posts

    def get_latest_links(self, count: int) -> List[str]:
        posts = self.get_posts_all(count)
        return [post.permalink for post in posts]

    @retry(delay=1, times=5)
    def upload_post(self, post: PostCreate) -> Response:
        return self._send(lambda: self.post_api.post_post_with_http_info(post=post))

    @retry(delay=1, times=5)
    def patch_post(self, id: int, patch: PostPatch) -> Optional[Response]:
        if not patch.to_dict():
            return None

        resp = self.post_api.post_id_patch_with_http_info(id, post=patch)
        return self._to_response(resp)

    @retry(delay=1, times=5)
    def get_films(self) -> List[Film]:
        return self.films_api.films_get().films or []

    @retry(delay=1, times=5)
    def get_cameras(self) -> List[Camera]:
        return self.cameras_api.cameras_get().cameras or []

    @retry(delay=1, times=5)
    def upload_film(self, film: FilmCreate) -> Response:
        return self._send(lambda: self.film_api.film_post_with_http_info(film=film))

    @retry(delay=1, times=5)
    def upload_camera(self, camera: CameraCreate) -> Response:
        return self._send(
            lambda: self.camera_api.camera_post_with_http_info(camera=camera)
        )

    def _send(self, call) -> Response:
        try:
            return self._to_response(call())
        except ApiException as e:
            return Response(status_code=e.status, text=e.body or "")

    def _to_response(self, resp: ApiResponse) -> Response:
        return Response(
            status_code=resp.status_code,
            text=resp.raw_data.decode("utf-8", errors="replace"),
        )

    def _filter_to_params(self, filter: Optional[PostsFilter]) -> Dict[str, Any]:
        params: Dict[str, Any] = {
            "page_size": DEFAULT_PAGE_SIZE,
            "sort": DEFAULT_SORT,
        }
        if filter is None:
            return params

        if (nsfw := filter.nsfw) is not None:
            params["nsfw"] = nsfw
        if (grayscale := filter.grayscale) is not None:
            params["grayscale"] = grayscale
        if (sprocket := filter.sprocket) is not None:
            params["sprocket"] = sprocket
        if (start := filter.time_start) is not None:
            params["time_start"] = start
        if (end := filter.time_end) is not None:
            params["time_end"] = end

        return params
