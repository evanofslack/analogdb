from dagster import ConfigurableResource
from analogdb.client import Client
from analogdb.scraping import Reddit
from typing import Optional


class AnalogDBResource(ConfigurableResource):
    base_url: str = "https://api.analogdb.com"
    username: Optional[str] = None
    password: Optional[str] = None

    def get_client(self) -> Client:
        return Client(
            base_url=self.base_url, username=self.username, password=self.password
        )


class RedditResource(ConfigurableResource):
    def get_client(self) -> Client:
        return Client()
