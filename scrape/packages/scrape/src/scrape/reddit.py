from typing import List, Optional

import praw
from praw.models import Comment
import requests
from PIL.Image import Image

from .constants import BW_SUB, REDDIT_URL, SPROCKET_SUB, VALID_CONTENT
from .image import ImageProcessor
from .models import RedditPost, ScrapeResult, ScrapeError, RedditComment


class RedditScrapingError(Exception):
    pass


class RedditScraper:
    def __init__(self, reddit: praw.Reddit, image_processor: ImageProcessor):
        self.reddit = reddit
        self.image = image_processor

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
                err = ScrapeError(id=submission.id, url=submission.url, msg=str(e))
                errors.append(err)
                continue

        result = ScrapeResult(posts=posts, errors=errors)
        return result

    def scrape_comments(self, url: str) -> List[RedditComment]:
        submission = self.reddit.submission(url=url)
        comments: List[RedditComment] = []

        # follow all comment trees
        submission.comments.replace_more(limit=None)

        # iterate over posts comments and convert to native type
        for c in submission.comments.list():
            try:
                if c is None:
                    continue

                if c is not Comment:
                    continue
                # deleted account's comments have text but no author
                if c.author.name is None:
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
            except Exception:
                continue

        return comments

    def _get_submissions(
        self, subreddit: str, num_posts: int, sort: str
    ) -> List[praw.reddit.Submission]:
        subreddit_obj = self.reddit.subreddit(subreddit)

        if sort == "new":
            submissions = subreddit_obj.new(limit=num_posts)
        elif sort == "top":
            submissions = subreddit_obj.top(limit=num_posts)
        else:
            submissions = subreddit_obj.hot(limit=num_posts)

        return [s for s in submissions if not s.is_self]

    def _process_submission(
        self,
        submission: praw.reddit.Submission,
        subreddit: str,
        latest_permalinks: List[str],
    ) -> Optional[RedditPost]:
        permalink = f"{REDDIT_URL}{submission.permalink}"
        if permalink in latest_permalinks:
            return None

        url = self._extract_url(submission)
        if not url:
            raise RedditScrapingError("No valid URL found")

        content_type = self._get_content_type(url)
        if content_type is None or content_type not in VALID_CONTENT:
            raise RedditScrapingError(f"Invalid content type: {content_type}")

        image = self._download_image(url)

        return RedditPost(
            image=image,
            width=image.width,
            height=image.height,
            content_type=content_type,
            title=submission.title,
            author=f"u/{submission.author.name}",
            permalink=permalink,
            score=submission.score,
            nsfw=submission.over_18,
            grayscale=self._is_grayscale(image, subreddit),
            time=int(submission.created_utc),
            sprocket=self._is_sprocket(subreddit),
        )

    def _extract_url(self, submission: praw.reddit.Submission) -> Optional[str]:
        if hasattr(submission, "is_gallery") and submission.is_gallery:
            return self._handle_gallery(submission)
        return submission.url

    def _handle_gallery(self, submission: praw.reddit.Submission) -> Optional[str]:
        for item in sorted(submission.gallery_data["items"], key=lambda x: x["id"]):
            media_id = item["media_id"]
            meta = submission.media_metadata[media_id]
            if meta["e"] == "Image":
                return meta["s"]["u"]
        return None

    def _get_content_type(self, url: str) -> Optional[str]:
        try:
            response = requests.head(url, stream=True)
            return response.headers.get("content-type")
        except Exception:
            return None

    def _validate_content(self, content_type: Optional[str]) -> bool:
        return content_type is not None and content_type in VALID_CONTENT

    def _download_image(self, url: str) -> Image:
        try:
            return self.image.download_image(url)
        except Exception as e:
            raise RedditScrapingError(f"Failed to download image from {url}: {e}")

    def _is_grayscale(self, image: Image, subreddit: str) -> bool:
        if subreddit == BW_SUB:
            return True
        return self.image.is_grayscale(image)

    def _is_sprocket(self, subreddit: str) -> bool:
        return subreddit == SPROCKET_SUB
