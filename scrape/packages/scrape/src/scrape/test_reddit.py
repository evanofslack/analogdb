from unittest.mock import MagicMock

import pytest
from praw.models import Comment, MoreComments

from .constants import REDDIT_URL
from .reddit import RedditScraper, RedditScrapingError


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
        scraper = RedditScraper(reddit)

        result = scraper.scrape_comments("https://reddit.com/r/analog/comments/abc")

        assert len(result) == 2
        assert result[0].author == "u/alice"
        assert result[0].body == "first"
        assert result[0].time == 1700000000
        assert result[0].permalink == f"{REDDIT_URL}/r/analog/comments/abc/first"
        assert result[1].author == "u/deleted"


START = 1_700_000_000
END = START + 86_400


def make_submission(id: str, score: int, **overrides) -> MagicMock:
    s = MagicMock()
    s.id = id
    s.score = score
    s.created_utc = START + 60
    s.removed_by_category = None
    s.is_self = False
    s.is_gallery = False
    s.author = MagicMock()
    s.author.name = "alice"
    s.url = f"https://i.redd.it/{id}.jpg"
    s.permalink = f"/r/analog/comments/{id}/title/"
    s.title = "title"
    s.selftext = ""
    s.over_18 = False
    for k, v in overrides.items():
        setattr(s, k, v)
    return s


def scrape_day(submissions, count=15, skip=()):
    reddit = MagicMock()
    reddit.info.return_value = iter(submissions)
    arctic = MagicMock()
    arctic.post_ids.return_value = [s.id for s in submissions]
    scraper = RedditScraper(reddit)
    result = scraper.scrape_day("analog", count, START, END, list(skip), arctic)
    return result, reddit


class TestScrapeDay:
    def test_looks_up_arctic_ids_and_ranks_by_score(self):
        subs = [make_submission("a", 5), make_submission("b", 50)]
        result, reddit = scrape_day(subs)

        reddit.info.assert_called_once_with(fullnames=["t3_a", "t3_b"])
        assert [p.score for p in result.posts] == [50, 5]
        assert result.posts[0].author == "u/alice"

    def test_skips_unusable_posts(self):
        subs = [
            make_submission("ok", 1),
            make_submission("removed", 9, removed_by_category="moderator"),
            make_submission("deleted", 9, author=None),
            make_submission("self", 9, is_self=True),
            make_submission("early", 9, created_utc=START - 1),
            make_submission("late", 9, created_utc=END),
        ]
        result, _ = scrape_day(subs)

        assert [p.permalink for p in result.posts] == [
            f"{REDDIT_URL}/r/analog/comments/ok/title/"
        ]
        assert result.errors == []

    def test_honours_skip_permalinks_and_count(self):
        subs = [make_submission(str(i), 100 - i) for i in range(5)]
        skip = [f"{REDDIT_URL}/r/analog/comments/0/title/"]
        result, _ = scrape_day(subs, count=2, skip=skip)

        assert [p.score for p in result.posts] == [99, 98]

    def test_keeps_going_past_a_bad_post(self):
        broken = make_submission(
            "broken", 99, is_gallery=True, gallery_data=None, media_metadata=None
        )
        result, _ = scrape_day([broken, make_submission("ok", 1)], count=1)

        assert len(result.posts) == 1
        assert [e.id for e in result.errors] == ["broken"]


class TestProcessSubmission:
    def test_deleted_author_is_a_scrape_error(self):
        scraper = RedditScraper(MagicMock())
        with pytest.raises(RedditScrapingError):
            scraper._process_submission(
                make_submission("a", 1, author=None), "analog", []
            )

    def test_gallery_without_media_has_no_url(self):
        scraper = RedditScraper(MagicMock())
        s = make_submission("a", 1, is_gallery=True, gallery_data=None)
        assert scraper._extract_url(s) is None

    def test_gallery_skips_failed_items(self):
        scraper = RedditScraper(MagicMock())
        s = make_submission(
            "a",
            1,
            is_gallery=True,
            gallery_data={
                "items": [{"id": 1, "media_id": "x"}, {"id": 2, "media_id": "y"}]
            },
            media_metadata={
                "x": {"status": "failed"},
                "y": {"e": "Image", "s": {"u": "https://i.redd.it/y.jpg"}},
            },
        )
        assert scraper._extract_url(s) == "https://i.redd.it/y.jpg"
