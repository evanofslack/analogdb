import praw
from dagster import ConfigurableResource
from analogdb.client import Client
from scrape.metadata import MetadataExtractor
from scrape.reddit import RedditScraper
from scrape.image import ImageProcessor
from typing import Optional
from openai import OpenAI


class AnalogDBResource(ConfigurableResource):
    base_url: str = "https://api.analogdb.com"
    username: Optional[str] = None
    password: Optional[str] = None

    def client(self) -> Client:
        return Client(
            base_url=self.base_url, username=self.username, password=self.password
        )


class RedditResource(ConfigurableResource):
    client_id: str = ""
    client_secret: str = ""
    user_agent: str = ""

    def client(self) -> RedditScraper:
        image_processor = ImageProcessor()
        reddit = praw.Reddit(
            client_id=self.client_id,
            client_secret=self.client_secret,
            user_agent=self.user_agent,
        )

        scraper = RedditScraper(reddit, image_processor)
        return scraper


class MetadataResource(ConfigurableResource):
    openai_url: str = ""
    openai_key: str = ""
    openai_model: str = ""

    def client(self) -> MetadataExtractor:
        ai = OpenAI(
            base_url=self.openai_url,
            api_key=self.openai_key,
        )
        extractor = MetadataExtractor(ai, self.openai_model)
        return extractor
