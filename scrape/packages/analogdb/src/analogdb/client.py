from enum import Enum
from importlib.metadata import version
from typing import Any, Dict, List, Optional, Tuple

from analogdb_generated import ApiClient, Configuration
from analogdb_generated.api.camera_api import CameraApi
from analogdb_generated.api.cameras_api import CamerasApi
from analogdb_generated.api.film_api import FilmApi
from analogdb_generated.api.films_api import FilmsApi
from analogdb_generated.api.post_api import PostApi
from analogdb_generated.api.posts_api import PostsApi
from analogdb_generated.exceptions import ApiException
from analogdb_generated.models.server_post_response import ServerPostResponse
from urllib3 import Retry

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
REQUEST_TIMEOUT = (10.0, 30.0)
RETRIES = Retry(
    total=4,
    backoff_factor=1,
    status_forcelist=[429, 502, 503, 504],
    allowed_methods=None,
    raise_on_status=False,
)


class Uploaded(str, Enum):
    CREATED = "created"
    EXISTS = "exists"


class Client:
    def __init__(
        self,
        base_url: str = "https://api.analogdb.com",
        username: Optional[str] = None,
        password: Optional[str] = None,
        timeout: Tuple[float, float] = REQUEST_TIMEOUT,
        retries: Retry = RETRIES,
    ):
        self.base_url = base_url
        self.timeout = timeout

        config = Configuration(host=f"{base_url}/v1")
        config.retries = retries
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
        cursor: Optional[str] = None,
    ) -> ServerPostResponse:
        params = self._filter_to_params(filter)
        params["page_size"] = count
        if cursor:
            params["cursor"] = cursor

        resp = self._call(self.posts_api.posts_get, **params)
        if resp.posts is None:
            resp.posts = []
        return resp

    def get_posts_all(
        self, count: int = 20, filter: Optional[PostsFilter] = None
    ) -> List[Post]:
        analog_posts = []
        cursor = None

        num = DEFAULT_PAGE_SIZE * 5
        if num > count:
            num = count
        while len(analog_posts) < count:
            resp = self.get_posts(num, filter, cursor)
            analog_posts.extend(resp.posts[: count - len(analog_posts)])
            cursor = resp.meta.next_cursor if resp.meta else None
            if not cursor:
                break

        return analog_posts

    def get_latest_links(self, count: int) -> List[str]:
        posts = self.get_posts_all(count)
        return [post.permalink for post in posts]

    def upload_post(self, post: PostCreate) -> Uploaded:
        try:
            self._call(self.post_api.post_post, post=post)
        except ApiException as e:
            if e.status == 409:
                return Uploaded.EXISTS
            raise
        return Uploaded.CREATED

    def patch_post(self, id: int, patch: PostPatch) -> None:
        if not patch.to_dict():
            return
        self._call(self.post_api.post_id_patch, id, post=patch)

    def get_films(self) -> List[Film]:
        return self._call(self.films_api.films_get).films or []

    def get_cameras(self) -> List[Camera]:
        return self._call(self.cameras_api.cameras_get).cameras or []

    def upload_film(self, film: FilmCreate) -> None:
        self._call(self.film_api.film_post, film=film)

    def upload_camera(self, camera: CameraCreate) -> None:
        self._call(self.camera_api.camera_post, camera=camera)

    def _call(self, fn, *args, **kwargs):
        return fn(*args, _request_timeout=self.timeout, **kwargs)

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
