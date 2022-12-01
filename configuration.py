import os
from dataclasses import dataclass
from functools import lru_cache


@dataclass
class Aws:
    access_key_id: str
    secret_access_key: str
    region_name: str


@dataclass
class Reddit:
    client_id: str
    client_secret: str
    user_agent: str


@dataclass
class Auth:
    username: str
    password: str


@dataclass
class Config:
    aws: Aws
    reddit: Reddit
    auth: Auth


@lru_cache(maxsize=None)
def init_config() -> Config:
    aws = Aws(
        access_key_id=os.getenv("AWS_ACCESS_KEY_ID"),
        secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY"),
        region_name=os.getenv("AWS_REGION"),
    )
    reddit = Reddit(
        client_id=os.getenv("client_id"),
        client_secret=os.getenv("client_secret"),
        user_agent=os.getenv("user_agent"),
    )

    auth = Auth(
        username=os.getenv("AUTH_USERNAME"),
        password=os.getenv("AUTH_PASSWORD"),
    )

    config = Config(aws=aws, reddit=reddit, auth=auth)

    return config
