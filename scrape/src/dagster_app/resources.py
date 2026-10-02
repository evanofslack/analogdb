import json
from typing import Any, Dict, List, Optional

import praw
from analogdb.client import Client
from dagster import ConfigurableResource
from dagster_aws.s3 import S3Resource
from openai import OpenAI
from scrape.image import ImageProcessor
from scrape.metadata import MetadataExtractor
from scrape.reddit import RedditScraper
from scrape.tagging import ImageTagger, catalog_words, load_stoplist

from .constants import DEFAULT_VISION_MODEL


class AnalogDBResource(ConfigurableResource):
    base_url: str = "https://api.analogdb.com"
    username: Optional[str] = None
    password: Optional[str] = None
    batch_posts_count: int = 100
    permalink_posts_count: int = 100

    def client(self) -> Client:
        return Client(
            base_url=self.base_url,
            username=self.username,
            password=self.password,
        )


class RedditResource(ConfigurableResource):
    client_id: str = ""
    client_secret: str = ""
    user_agent: str = ""

    def client(self) -> RedditScraper:
        reddit = praw.Reddit(
            client_id=self.client_id,
            client_secret=self.client_secret,
            user_agent=self.user_agent,
        )

        scraper = RedditScraper(reddit)
        return scraper


class ImageProcessorResource(ConfigurableResource):
    def client(self) -> ImageProcessor:
        return ImageProcessor()


class MetadataResource(ConfigurableResource):
    openai_url: str = ""
    openai_key: str = ""
    openai_model: str = "google/gemini-2.5-flash-lite"
    batch_size: int = 20

    def client(self) -> MetadataExtractor:
        ai = OpenAI(
            base_url=self.openai_url,
            api_key=self.openai_key,
        )
        return MetadataExtractor(ai, self.openai_model, self.batch_size)


class TaggerResource(ConfigurableResource):
    openai_url: str = ""
    openai_key: str = ""
    openai_model: str = DEFAULT_VISION_MODEL
    concurrency: int = 8
    stoplist_path: str = ""

    def client(
        self, cameras: List[Dict[str, Any]], films: List[Dict[str, Any]]
    ) -> ImageTagger:
        ai = OpenAI(
            base_url=self.openai_url,
            api_key=self.openai_key,
        )
        return ImageTagger(
            ai,
            self.openai_model,
            catalog_words(cameras, films),
            load_stoplist(self.stoplist_path),
        )


class StorageResource(ConfigurableResource):
    s3_resource: S3Resource

    def get_object(self, bucket: str, key: str) -> Optional[bytes]:
        """File data, or None when the key doesn't exist."""
        s3 = self.s3_resource.get_client()
        try:
            return s3.get_object(Bucket=bucket, Key=key)["Body"].read()
        except s3.exceptions.NoSuchKey:
            return None

    def put_object(self, bucket: str, key: str, body: bytes, content_type: str) -> None:
        """Upload file data. The caller builds the public CloudFront URL."""
        self.s3_resource.get_client().put_object(
            Bucket=bucket, Key=key, Body=body, ContentType=content_type
        )


class FilmsJsonResource(ConfigurableResource):
    file_path: str

    def client(self) -> List[Dict[str, Any]]:
        with open(self.file_path, "r") as f:
            return json.load(f)


class CamerasJsonResource(ConfigurableResource):
    file_path: str

    def client(self) -> List[Dict[str, Any]]:
        with open(self.file_path, "r") as f:
            return json.load(f)
