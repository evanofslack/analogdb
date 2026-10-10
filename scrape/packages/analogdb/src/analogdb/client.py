from enum import Enum
from importlib.metadata import version
from typing import Any, Dict, Iterator, List, Optional, Tuple

from analogdb_generated import ApiClient, Configuration
from analogdb_generated.api.camera_api import CameraApi
from analogdb_generated.api.cameras_api import CamerasApi
from analogdb_generated.api.extractions_api import ExtractionsApi
from analogdb_generated.api.film_api import FilmApi
from analogdb_generated.api.films_api import FilmsApi
from analogdb_generated.api.post_api import PostApi
from analogdb_generated.api.posts_api import PostsApi
from analogdb_generated.api.removed_api import RemovedApi
from analogdb_generated.api.scrape_api import ScrapeApi
from analogdb_generated.api.similarity_api import SimilarityApi
from analogdb_generated.exceptions import ApiException
from analogdb_generated.models.server_encode_posts_request import (
    ServerEncodePostsRequest,
)
from analogdb_generated.models.server_extractions_request import (
    ServerExtractionsRequest,
)
from analogdb_generated.models.server_post_response import ServerPostResponse
from urllib3 import Retry

from .models import (
    Camera,
    CameraCreate,
    Film,
    FilmCreate,
    Post,
    PostCreate,
    PostExtraction,
    PostPatch,
    PostsFilter,
)

DEFAULT_PAGE_SIZE = 20
DEFAULT_SORT = "time"
USER_AGENT = f"analogdb-scraper/{version('analogdb')}"
REQUEST_TIMEOUT = (10.0, 30.0)
ENCODE_TIMEOUT = (10.0, 900.0)
EXTRACTIONS_CHUNK = 200
EXTRACTIONS_PAGE_SIZE = 500
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


class Deleted(str, Enum):
    DELETED = "deleted"
    IN_USE = "in_use"
    MISSING = "missing"


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
        self.extractions_api = ExtractionsApi(self.api_client)
        self.removed_api = RemovedApi(self.api_client)
        self.scrape_api = ScrapeApi(self.api_client)
        self.similarity_api = SimilarityApi(self.api_client)

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

    def get_removed_links(self) -> List[str]:
        """Permalinks of deleted posts, which must never be scraped again"""
        return (
            self._call(self.removed_api.admin_removed_permalinks_get).permalinks or []
        )

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

    def get_post_ids(self) -> List[int]:
        return self._call(self.posts_api.ids_get).ids or []

    def encode_posts(self, ids: List[int], batch_size: int = 20) -> List[int]:
        """Encode the posts' image vectors, returning the ids that failed. The
        route takes at most 100 per batch and has 15 minutes per request."""
        if not ids:
            return []
        resp = self.similarity_api.encode_put(
            request=ServerEncodePostsRequest(ids=ids, batch_size=batch_size),
            _request_timeout=ENCODE_TIMEOUT,
        )
        return resp.failed_ids or []

    def get_missing_captions(self, version: Optional[str] = None) -> List[int]:
        """Ids of posts with no caption, or with a caption of another version."""
        return (
            self._call(self.scrape_api.scrape_captions_missing_get, version=version).ids
            or []
        )

    def get_missing_vectors(self) -> List[int]:
        """Ids of posts with no image vector, oldest first."""
        return self._call(self.scrape_api.scrape_vectors_missing_get).ids or []

    def get_films(self) -> List[Film]:
        return self._call(self.films_api.films_get).films or []

    def get_cameras(self) -> List[Camera]:
        return self._call(self.cameras_api.cameras_get).cameras or []

    def upload_film(self, film: FilmCreate) -> None:
        self._call(self.film_api.film_post, film=film)

    def upload_camera(self, camera: CameraCreate) -> None:
        self._call(self.camera_api.camera_post, camera=camera)

    def delete_film(self, id: int) -> Deleted:
        return self._delete(self.film_api.film_id_delete, id)

    def delete_camera(self, id: int) -> Deleted:
        return self._delete(self.camera_api.camera_id_delete, id)

    def _delete(self, fn, id: int) -> Deleted:
        """Delete a catalog entry. The API refuses one that posts still use."""
        try:
            self._call(fn, id)
        except ApiException as e:
            if e.status == 409:
                return Deleted.IN_USE
            if e.status == 404:
                return Deleted.MISSING
            raise
        return Deleted.DELETED

    def upsert_extractions(
        self, extractions: List[PostExtraction], chunk_size: int = EXTRACTIONS_CHUNK
    ) -> Tuple[int, List[int]]:
        """Store extractions, returning how many were written and the post ids
        skipped because the post no longer exists."""
        written = 0
        skipped: List[int] = []
        for start in range(0, len(extractions), chunk_size):
            chunk = extractions[start : start + chunk_size]
            resp = self._call(
                self.extractions_api.admin_extractions_post,
                extractions=ServerExtractionsRequest(extractions=chunk),
            )
            written += resp.written or 0
            skipped.extend(resp.skipped or [])
        return written, skipped

    def get_extractions(
        self,
        has_unmatched: Optional[bool] = None,
        kind: Optional[str] = None,
        key: Optional[str] = None,
        full: bool = False,
        before_id: Optional[int] = None,
        limit: int = EXTRACTIONS_PAGE_SIZE,
    ) -> Tuple[List[PostExtraction], Optional[int]]:
        resp = self._call(
            self.extractions_api.admin_extractions_get,
            has_unmatched=has_unmatched,
            kind=kind,
            key=key,
            full=full,
            before_id=before_id,
            limit=limit,
        )
        return resp.extractions or [], resp.next_before_id

    def iter_extractions(self, **filters: Any) -> Iterator[PostExtraction]:
        before_id = None
        while True:
            page, before_id = self.get_extractions(before_id=before_id, **filters)
            yield from page
            if before_id is None:
                return

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
