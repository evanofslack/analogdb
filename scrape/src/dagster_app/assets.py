import dagster as dg
from analogdb.models import Post
from typing import List
from .resources import AnalogDBResource


@dg.asset
def posts(analogdb: AnalogDBResource) -> List[Post]:
    posts = analogdb.get_posts_all(count=100)

    dg.get_dagster_logger().info(f"Fetched {len(posts)} posts")

    return posts
