from unittest.mock import MagicMock

from praw.models import Comment, MoreComments

from .constants import REDDIT_URL
from .reddit import RedditScraper


def make_comment(author: str | None, body: str) -> MagicMock:
    c = MagicMock(spec=Comment)
    if author is None:
        c.author = None
    else:
        c.author = MagicMock()
        c.author.name = author
    c.body = body
    c.score = 3
    c.created_utc = 1700000000.5
    c.permalink = f"/r/analog/comments/abc/{body}"
    return c


class TestScrapeComments:
    def test_skips_more_comments_and_handles_deleted_author(self):
        comments = [
            make_comment("alice", "first"),
            MagicMock(spec=MoreComments),
            make_comment(None, "second"),
        ]
        reddit = MagicMock()
        reddit.submission.return_value.comments.list.return_value = comments
        scraper = RedditScraper(reddit, MagicMock())

        result = scraper.scrape_comments("https://reddit.com/r/analog/comments/abc")

        assert len(result) == 2
        assert result[0].author == "u/alice"
        assert result[0].body == "first"
        assert result[0].time == 1700000000
        assert result[0].permalink == f"{REDDIT_URL}/r/analog/comments/abc/first"
        assert result[1].author == "u/deleted"
