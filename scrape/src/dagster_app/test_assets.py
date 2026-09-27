from unittest.mock import MagicMock, patch

import analogdb.models as adb
import dagster as dg
import pytest

from .assets import updated_reddit_comments
from .resources import RedditResource


def run_updated_reddit_comments(post_count: int):
    posts = [
        adb.Post(
            id=i,
            title="title",
            author="u/author",
            permalink=f"/r/analog/{i}",
            description=None,
            score=1,
            timestamp=0,
            nsfw=False,
            grayscale=False,
            sprocket=False,
            images=[],
        )
        for i in range(post_count)
    ]
    scraper = MagicMock()
    scraper.scrape_comments.return_value = []
    context = dg.build_asset_context(partition_key="2024-01-01")
    with patch.object(RedditResource, "client", return_value=scraper):
        return updated_reddit_comments(context, posts, RedditResource())


class TestUpdatedRedditComments:
    def test_fails_when_many_posts_have_no_comments(self):
        with pytest.raises(dg.Failure):
            run_updated_reddit_comments(5)

    def test_allows_few_posts_with_no_comments(self):
        result = run_updated_reddit_comments(3)
        assert len(result) == 3
