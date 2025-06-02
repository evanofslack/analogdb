from dataclasses import dataclass, fields
from typing import List, Optional, Set

import boto3
import praw
import openai
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
    greyscale: bool
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
class AnalogKeyword:
    word: str
    weight: float


@dataclass
class Color:
    hex: str
    css: str
    html: str
    percent: float


@dataclass
class AnalogDisplayPost:
    """
    post model as returned from analogdb api

    """

    id: int
    title: str
    author: str
    permalink: str
    score: int
    nsfw: bool
    grayscale: bool
    timestamp: float
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
    raw_url: str
    raw_width: int
    raw_height: int


@dataclass
class CloudfrontImage:
    url: str
    width: int
    height: int


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
class PatchPost:
    score: Optional[int]
    nsfw: Optional[bool]
    greyscale: Optional[bool]
    sprocket: Optional[bool]
    colors: Optional[List[Color]]
    keywords: Optional[List[AnalogKeyword]]
    metadata: Optional[PhotoMetadata]

    def is_empty(self) -> bool:
        return all(getattr(self, field.name) is None for field in fields(self))


@dataclass
class AnalogPost:
    url: str
    title: str
    author: str
    permalink: str
    score: int
    nsfw: bool
    greyscale: bool
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

    keywords: List[AnalogKeyword]
    colors: List[Color]


def new_analog_post(
    images: List[CloudfrontImage],
    post: RedditPost,
    keywords: List[AnalogKeyword],
    colors: List[Color],
    metadata: PhotoMetadata,
) -> AnalogPost:
    low_img = images[0]
    med_img = images[1]
    high_img = images[2]
    raw_img = images[3]

    analog_post = AnalogPost(
        url=raw_img.url,
        title=post.title,
        author=post.author,
        permalink=post.permalink,
        score=post.score,
        nsfw=post.nsfw,
        greyscale=post.greyscale,
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

    return analog_post


@dataclass
class AwsCreds:
    access_key_id: str
    secret_access_key: str
    region_name: str


@dataclass
class RedditCreds:
    client_id: str
    client_secret: str
    user_agent: str


@dataclass
class OpenAICreds:
    key: str
    url: str


@dataclass
class AuthCreds:
    username: str
    password: str


@dataclass
class SlackWebhook:
    url: str


@dataclass
class App:
    log_level: str
    env: str
    api_base_url: str
    openai_model: str


@dataclass
class Config:
    aws: AwsCreds
    reddit: RedditCreds
    openai: OpenAICreds
    auth: AuthCreds
    slack: SlackWebhook
    app: App


@dataclass
class Dependencies:
    cfg: Config
    s3_client: boto3.session.Session
    reddit_client: praw.Reddit
    auth: AuthCreds
    openai: openai.OpenAI
    blacklist: Set[str]
