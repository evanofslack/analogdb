from dataclasses import dataclass
from typing import Optional

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
    time: float
    sprocket: bool


@dataclass
class AnalogPost:
    url: str
    title: str
    author: str
    permalink: str
    score: int
    nsfw: bool
    greyscale: bool
    time: float
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
class Color:
    hex: str
    css: str
    percent: int


@dataclass
class PatchPost:
    score: Optional[int]
    nsfw: Optional[bool]
    greyscale: Optional[bool]
    sprocket: Optional[bool]


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
class Config:
    aws: AwsCreds
    reddit: RedditCreds
    auth: AuthCreds
    slack: SlackWebhook


@dataclass
class Dependencies:
    s3_client: boto3.session.Session
    reddit_client: praw.Reddit
    auth: AuthCreds
