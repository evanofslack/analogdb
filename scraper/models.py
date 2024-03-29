from dataclasses import dataclass
from typing import List, Optional, Set

import boto3
import praw
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
    keywords: List[AnalogKeyword]
    colors: List[Color]


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
class PatchPost:
    score: Optional[int]
    nsfw: Optional[bool]
    greyscale: Optional[bool]
    sprocket: Optional[bool]
    colors: Optional[List[Color]]
    keywords: Optional[List[AnalogKeyword]]


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


@dataclass
class Config:
    aws: AwsCreds
    reddit: RedditCreds
    auth: AuthCreds
    slack: SlackWebhook
    app: App


@dataclass
class Dependencies:
    s3_client: boto3.session.Session
    reddit_client: praw.Reddit
    auth: AuthCreds
    blacklist: Set[str]
