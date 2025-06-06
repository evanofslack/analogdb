from dataclasses import dataclass
from typing import List, Tuple

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
