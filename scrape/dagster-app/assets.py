import dagster as dg
from analogdb.client import Client
from analogdb.models import Post
from typing import List


@dg.asset
def posts() -> List[Post]:
    client = Client()
    posts = client.get_posts_all(count=100)

    dg.get_dagster_logger().info(f"Fetched {len(posts)} posts")

    return posts


defs = dg.Definitions(assets=[posts])
