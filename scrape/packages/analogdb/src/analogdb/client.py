import functools
import json
import time
from typing import Any, Dict, List, Optional

import requests
from requests.auth import HTTPBasicAuth

from .models import (
    Camera,
    CameraCreate,
    Film,
    FilmCreate,
    Image,
    Meta,
    Post,
    PostCreate,
    PostPatch,
    Posts,
    PostsFilter,
)

DEFAULT_PAGE_SIZE = 20
DEFAULT_SORT = "latest"


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


class Client:
    def __init__(
        self,
        base_url: str = "https://api.analogdb.com",
        username: Optional[str] = None,
        password: Optional[str] = None,
    ):
        self.base_url = base_url
        self.session = requests.Session()
        self.auth = HTTPBasicAuth(username, password) if username and password else None

    def get_post(self, post_id: int) -> Post:
        response = self.session.get(f"{self.base_url}/post/{post_id}")
        response.raise_for_status()

        data = response.json()
        return self._parse_post(data)

    def get_posts(
        self,
        count: int = DEFAULT_PAGE_SIZE,
        filter: Optional[PostsFilter] = None,
        page_id: Optional[int] = None,
    ) -> Posts:
        params = self._filter_to_params(filter)
        params["page_size"] = count
        if page_id is not None:
            params["page_id"] = page_id

        url = f"{self.base_url}/posts"
        response = self.session.get(url, params=params)
        response.raise_for_status()

        data = response.json()
        return self._parse_posts_response(data)

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
            if resp.meta.next_page_url == "":
                break
            page_id = resp.meta.next_page_id

        return analog_posts

    def get_latest_links(self, count: int) -> List[str]:
        posts = self.get_posts_all(count)
        return [post.permalink for post in posts]

    @retry(delay=1, times=5)
    def upload_post(self, post: PostCreate) -> requests.Response:
        json_post = json.dumps(post.to_json())
        response = self.session.put(
            f"{self.base_url}/post", data=json_post, auth=self.auth
        )
        return response

    @retry(delay=1, times=5)
    def patch_post(self, patch: PostPatch) -> Optional[requests.Response]:
        if patch.is_empty():
            return None

        json_patch = json.dumps(patch.to_json())
        response = self.session.patch(
            f"{self.base_url}/post/{patch.id}", data=json_patch, auth=self.auth
        )
        response.raise_for_status()
        return response

    @retry(delay=1, times=5)
    def delete_post(self, post_id: int) -> requests.Response:
        response = self.session.delete(
            f"{self.base_url}/post/{post_id}", auth=self.auth
        )
        response.raise_for_status()
        return response

    def get_all_post_ids(self) -> List[int]:
        response = self.session.get(f"{self.base_url}/ids")
        response.raise_for_status()

        data = response.json()
        return [int(id) for id in data["ids"]]

    @retry(delay=1, times=5)
    def get_keyword_updated_post_ids(self) -> List[int]:
        response = self.session.get(
            f"{self.base_url}/scrape/keywords/updated", auth=self.auth
        )
        response.raise_for_status()

        data = response.json()
        return data["ids"]

    @retry(delay=1, times=5)
    def get_films(
        self,
    ) -> List[Film]:
        url = f"{self.base_url}/films"
        response = self.session.get(url)
        response.raise_for_status()
        data = response.json()
        return [self._parse_film(film_data) for film_data in data["films"]]

    @retry(delay=1, times=5)
    def get_cameras(self) -> List[Camera]:
        url = f"{self.base_url}/cameras"
        response = self.session.get(url)
        response.raise_for_status()
        data = response.json()
        return [self._parse_camera(camera_data) for camera_data in data["cameras"]]

    @retry(delay=1, times=5)
    def upload_film(self, film: FilmCreate) -> requests.Response:
        json_film = json.dumps(film.to_json())
        response = self.session.put(
            f"{self.base_url}/films", data=json_film, auth=self.auth
        )
        return response

    @retry(delay=1, times=5)
    def upload_camera(self, camera: CameraCreate) -> requests.Response:
        json_camera = json.dumps(camera.to_json())
        response = self.session.put(
            f"{self.base_url}/cameras", data=json_camera, auth=self.auth
        )
        return response

    def encode_images(self, ids: List[int], batch_size: int) -> requests.Response:
        data = {"ids": ids, "batch_size": batch_size}
        body = json.dumps(data)

        response = self.session.put(
            f"{self.base_url}/encode", data=body, auth=self.auth
        )
        response.raise_for_status()
        return response

    def _parse_posts_response(self, data: Dict[str, Any]) -> Posts:
        posts = [self._parse_post(post_data) for post_data in data.get("posts") or []]
        meta = self._parse_meta(data["meta"])
        return Posts(posts=posts, meta=meta)

    def _parse_meta(self, data: Dict[str, Any]) -> Meta:
        return Meta(
            total_posts=data["total_posts"],
            page_size=data["page_size"],
            next_page_id=data["next_page_id"],
            next_page_url=data["next_page_url"],
        )

    def _parse_post(self, data: Dict[str, Any]) -> Post:
        images = [Image(**img) for img in data["images"]]
        return Post(
            id=data["id"],
            title=data["title"],
            author=data["author"],
            permalink=data["permalink"],
            description=data.get("description"),  # may be none
            score=data["score"],
            timestamp=data["timestamp"],
            nsfw=data["nsfw"],
            grayscale=data["grayscale"],
            sprocket=data["sprocket"],
            images=images,
        )

    def _parse_film(self, data: Dict[str, Any]) -> Film:
        return Film(
            id=data["id"],
            type=data["type"],
            make=data["make"],
            speed=data["speed"],
            color_type=data["color_type"],
            description=data["description"],
        )

    def _parse_camera(self, data: Dict) -> Camera:
        return Camera(
            id=str(data["id"]),
            make=data["make"],
            model=data["model"],
            description=data["description"],
        )

    def _filter_to_params(self, filter: Optional[PostsFilter]) -> Dict[str, int | str]:
        params: Dict[str, str | int] = {
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


if __name__ == "__main__":
    c = Client()
    posts = c.get_posts(count=20)
    print(posts)
