from typing import Collection, List, Optional

import praw
from praw.models import Comment

from .arctic import ArcticShift
from .constants import REDDIT_URL, SPROCKET_SUB
from .models import RedditComment, RedditPost, ScrapeError, ScrapeResult


class RedditScrapingError(Exception):
    pass


class RedditScraper:
    def __init__(self, reddit: praw.Reddit):
        self.reddit = reddit

    def scrape_posts(
        self,
        subreddit: str,
        num_posts: int,
        latest_permalinks: List[str],
        sort: str = "hot",
    ) -> ScrapeResult:
        submissions = self._get_submissions(subreddit, num_posts, sort)

        posts: List[RedditPost] = []
        errors: List[ScrapeError] = []
        for submission in submissions:
            try:
                post = self._process_submission(
                    submission, subreddit, latest_permalinks
                )
                if post:
                    posts.append(post)
            except RedditScrapingError as e:
                err = ScrapeError(
                    id=submission.id,
                    url=submission.url,
                    permalink=f"{REDDIT_URL}{submission.permalink}",
                    msg=str(e),
                )
                errors.append(err)
                continue

        result = ScrapeResult(posts=posts, errors=errors)
        return result

    def scrape_day(
        self,
        subreddit: str,
        num_posts: int,
        start: int,
        end: int,
        skip_permalinks: List[str],
        arctic: ArcticShift,
    ) -> ScrapeResult:
        """Top posts by current score created in [start, end), found through
        Arctic Shift since Reddit listings cannot be searched by time"""
        ids = arctic.post_ids(subreddit, start, end)
        submissions = list(self.reddit.info(fullnames=[f"t3_{id}" for id in ids]))
        usable = [s for s in submissions if self._is_usable(s, start, end)]
        usable.sort(key=lambda s: s.score, reverse=True)

        skip = set(skip_permalinks)
        posts: List[RedditPost] = []
        errors: List[ScrapeError] = []
        for submission in usable:
            if len(posts) >= num_posts:
                break
            try:
                post = self._process_submission(submission, subreddit, skip)
                if post:
                    posts.append(post)
            except RedditScrapingError as e:
                errors.append(
                    ScrapeError(
                        id=submission.id,
                        url=submission.url,
                        permalink=f"{REDDIT_URL}{submission.permalink}",
                        msg=str(e),
                    )
                )

        return ScrapeResult(posts=posts, errors=errors)

    def _is_usable(self, submission: praw.reddit.Submission, start: int, end: int):
        return (
            not getattr(submission, "removed_by_category", None)
            and submission.author is not None
            and not submission.is_self
            and start <= submission.created_utc < end
        )

    def scrape_comments(self, url: str) -> List[RedditComment]:
        submission = self.reddit.submission(url=url)
        comments: List[RedditComment] = []

        # follow all comment trees
        submission.comments.replace_more(limit=None)

        # iterate over posts comments and convert to native type
        for c in submission.comments.list():
            if not isinstance(c, Comment):
                continue

            # deleted account's comments have text but no author
            if c.author is None:
                author = "deleted"
            else:
                author = c.author.name

            comment = RedditComment(
                body=c.body,
                score=c.score,
                author=f"u/{author}",
                time=int(c.created_utc),
                permalink=f"{REDDIT_URL}{c.permalink}",
            )
            comments.append(comment)

        return comments

    def updated_score(self, url: str, score: int) -> int | None:
        try:
            submission = self.reddit.submission(url=url)
            new_score = submission.score
        except Exception:
            return None

        # only update the score if the new score is higher than original
        if new_score > score:
            return new_score

    def updated_selftext(self, url: str) -> str | None:
        try:
            submission = self.reddit.submission(url=url)
            text = submission.selftext
        except Exception:
            return None

        return text

    def _get_submissions(
        self, subreddit: str, num_posts: int, sort: str
    ) -> List[praw.reddit.Submission]:
        sub = self.reddit.subreddit(subreddit)

        if sort == "new":
            submissions = sub.new(limit=num_posts)
        elif sort == "top":
            submissions = sub.top(limit=num_posts, time_filter="day")
        else:
            submissions = sub.hot(limit=num_posts)

        return [s for s in submissions if not s.is_self]

    def _process_submission(
        self,
        submission: praw.reddit.Submission,
        subreddit: str,
        latest_permalinks: Collection[str],
    ) -> Optional[RedditPost]:
        permalink = f"{REDDIT_URL}{submission.permalink}"
        if permalink in latest_permalinks:
            return None
        if submission.author is None:
            raise RedditScrapingError("Author deleted")

        url = self._extract_url(submission)
        if not url:
            raise RedditScrapingError("No valid URL found")

        return RedditPost(
            image_url=url,
            subreddit=subreddit,
            title=submission.title,
            selftext=submission.selftext,
            author=f"u/{submission.author.name}",
            permalink=permalink,
            score=submission.score,
            nsfw=submission.over_18,
            time=int(submission.created_utc),
            sprocket=self._is_sprocket(subreddit),
        )

    def _extract_url(self, submission: praw.reddit.Submission) -> Optional[str]:
        if hasattr(submission, "is_gallery") and submission.is_gallery:
            return self._handle_gallery(submission)
        return submission.url

    def _handle_gallery(self, submission: praw.reddit.Submission) -> Optional[str]:
        # galleries removed or deleted later keep is_gallery but lose their media
        if not getattr(submission, "gallery_data", None) or not getattr(
            submission, "media_metadata", None
        ):
            return None
        for item in sorted(submission.gallery_data["items"], key=lambda x: x["id"]):
            media_id = item["media_id"]
            meta = submission.media_metadata.get(media_id, {})
            if meta.get("e") == "Image" and "s" in meta:
                return meta["s"]["u"]
        return None

    def _is_sprocket(self, subreddit: str) -> bool:
        return subreddit == SPROCKET_SUB
