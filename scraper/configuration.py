import os
from functools import lru_cache
from typing import Set

import boto3
import praw
from dotenv import load_dotenv
from loguru import logger

from constants import BLACKLIST_KEYWORDS_PATH
from models import (App, AuthCreds, AwsCreds, Config, Dependencies,
                    RedditCreds, SlackWebhook)


@lru_cache(maxsize=None)
def init_config() -> Config:

    load_dotenv()

    aws = AwsCreds(
        access_key_id=os.getenv("AWS_ACCESS_KEY_ID"),
        secret_access_key=os.getenv("AWS_SECRET_ACCESS_KEY"),
        region_name=os.getenv("AWS_REGION"),
    )
    reddit = RedditCreds(
        client_id=os.getenv("REDDIT_CLIENT_ID"),
        client_secret=os.getenv("REDDIT_CLIENT_SECRET"),
        user_agent=os.getenv("REDDIT_USER_AGENT"),
    )

    auth = AuthCreds(
        username=os.getenv("AUTH_USERNAME"),
        password=os.getenv("AUTH_PASSWORD"),
    )

    slack = SlackWebhook(
        url=os.getenv("SLACK_WEBHOOK_URL"),
    )

    app = App(
        log_level=os.getenv("LOG_LEVEL"),
        env=os.getenv("APP_ENV"),
        api_base_url=os.getenv("API_BASE_URL"),
    )

    config = Config(aws=aws, reddit=reddit, auth=auth, slack=slack, app=app)

    return config


def init_s3_client(creds: AwsCreds) -> boto3.session.Session:
    s3 = boto3.client(
        "s3",
        aws_access_key_id=creds.access_key_id,
        aws_secret_access_key=creds.secret_access_key,
        region_name=creds.region_name,
    )
    return s3


def init_reddit_client(creds: RedditCreds) -> praw.Reddit:
    reddit = praw.Reddit(
        client_id=creds.client_id,
        client_secret=creds.client_secret,
        user_agent=creds.user_agent,
    )
    return reddit


def load_blacklist(filepath) -> Set[str]:

    with open(filepath, "r") as f:
        lines = f.read().splitlines()

    return set(lines)


def dependencies_from_config(config: Config) -> Dependencies:
    deps = Dependencies(
        s3_client=init_s3_client(creds=config.aws),
        blacklist=load_blacklist(filepath=BLACKLIST_KEYWORDS_PATH),
        reddit_client=init_reddit_client(creds=config.reddit),
        auth=config.auth,
    )
    return deps
