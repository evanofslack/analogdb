import json
from typing import Any, Dict, List, Optional

import praw
from analogdb.client import Client
from dagster import ConfigurableResource
from dagster_aws.s3 import S3Resource
from openai import OpenAI
from scrape.image import ImageProcessor
from scrape.keywords import KeywordBlacklist, KeywordExtractor
from scrape.metadata import MetadataExtractor
from scrape.reddit import RedditScraper


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
    openai_model: str = "google/gemini-2.5-flash"
    batch_size: int = 25

    def client(self) -> MetadataExtractor:
        ai = OpenAI(
            base_url=self.openai_url,
            api_key=self.openai_key,
        )
        extractor = MetadataExtractor(ai, self.openai_model, self.batch_size)
        return extractor


class StorageResource(ConfigurableResource):
    s3_resource: S3Resource

    def put_object(self, bucket: str, key: str, body: bytes, content_type: str) -> None:
        """Upload file data. The caller builds the public CloudFront URL."""
        self.s3_resource.get_client().put_object(
            Bucket=bucket, Key=key, Body=body, ContentType=content_type
        )


class KeywordExtractorResource(ConfigurableResource):
    max_keywords: int

    def client(self) -> KeywordExtractor:
        return KeywordExtractor()


class KeywordBlacklistResource(ConfigurableResource):
    file_path: str

    def client(self) -> KeywordBlacklist:
        return KeywordBlacklist(self.file_path)


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
