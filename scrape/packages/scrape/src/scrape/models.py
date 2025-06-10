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
