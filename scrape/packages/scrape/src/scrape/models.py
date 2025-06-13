from dataclasses import dataclass, fields
from typing import List, Optional

from PIL.Image import Image


@dataclass
class RedditPost:
    image: Image
    width: int
    height: int
    content_type: str
    title: str
    author: str
    permalink: str
    score: int
    nsfw: bool
    grayscale: bool
    time: int
    sprocket: bool


@dataclass
class RedditComment:
    body: str
    score: int
    author: str
    time: int
    permalink: str


@dataclass
class ScrapeError:
    id: str
    url: str
    msg: str


@dataclass
class ScrapeResult:
    posts: List[RedditPost]
    errors: List[ScrapeError]


@dataclass
class Color:
    hex: str
    css: str
    html: str
    percent: float


@dataclass
class PhotoMetadata:
    camera_make: Optional[str] = None
    camera_model: Optional[str] = None
    film_make: Optional[str] = None
    film_type: Optional[str] = None
    film_speed: Optional[int] = None
    focal_length: Optional[int] = None
    aperture: Optional[str] = None

    def is_empty(self) -> bool:
        return all(getattr(self, field.name) is None for field in fields(self))


@dataclass
class Keyword:
    word: str
    weight: float


@dataclass
class S3Image:
    url: str
    width: int
    height: int


@dataclass
class UploadPost:
    url: str
    title: str
    author: str
    permalink: str
    score: int
    nsfw: bool
    grayscale: bool
    time: int
    width: int
    height: int
    sprocket: bool

    low_url: str
    low_width: int
    low_height: int
    med_url: str
    med_width: int
    med_height: int
    high_url: str
    high_width: int
    high_height: int

    camera_make: Optional[str]
    camera_model: Optional[str]
    film_make: Optional[str]
    film_type: Optional[str]
    film_speed: Optional[int]
    focal_length: Optional[int]
    aperture: Optional[str]

    keywords: List[Keyword]
    colors: List[Color]


def create_upload_post(
    post: RedditPost,
    metadata: PhotoMetadata,
    images: List[S3Image],
    keywords: List[Keyword],
    colors: List[Color],
) -> UploadPost | None:
    if len(images) < 4:
        return None

    low_img = images[0]
    med_img = images[1]
    high_img = images[2]
    raw_img = images[3]

    up = UploadPost(
        url=raw_img.url,
        title=post.title,
        author=post.author,
        permalink=post.permalink,
        score=post.score,
        nsfw=post.nsfw,
        grayscale=post.grayscale,
        time=post.time,
        width=raw_img.width,
        height=raw_img.height,
        sprocket=post.sprocket,
        low_url=low_img.url,
        low_width=low_img.width,
        low_height=low_img.height,
        med_url=med_img.url,
        med_width=med_img.width,
        med_height=med_img.height,
        high_url=high_img.url,
        high_width=high_img.width,
        high_height=high_img.height,
        camera_make=metadata.camera_make,
        camera_model=metadata.camera_model,
        film_make=metadata.film_make,
        film_type=metadata.film_type,
        film_speed=metadata.film_speed,
        focal_length=metadata.focal_length,
        aperture=metadata.aperture,
        keywords=keywords,
        colors=colors,
    )
    return up
