"""Post ids for a subreddit and time window from the Arctic Shift archive.

Reddit has no time range search, so backfills find the ids of a missed day
here and look the posts up through the Reddit API for current scores and
removal status.
"""

import time
from typing import Callable, List, Optional

import requests

from .constants import USER_AGENT

ARCTIC_SHIFT_URL = "https://arctic-shift.photon-reddit.com"
PAGE_SIZE = 100
MAX_TRIES = 5
RETRY_STATUS = {422, 429, 500, 502, 503, 504}
TIMEOUT = (10, 30)


class ArcticShiftError(Exception):
    pass


class ArcticShift:
    def __init__(
        self,
        session: Optional[requests.Session] = None,
        base_url: str = ARCTIC_SHIFT_URL,
        sleep: Callable[[float], None] = time.sleep,
    ):
        self.session = session or requests.Session()
        self.session.headers["User-Agent"] = USER_AGENT
        self.base_url = base_url
        self.sleep = sleep

    def post_ids(self, subreddit: str, start: int, end: int) -> List[str]:
        """Ids of posts created in [start, end), oldest first"""
        ids: List[str] = []
        after = start
        while True:
            page = self._search(subreddit, after, end)
            ids += [p["id"] for p in page]
            if len(page) < PAGE_SIZE:
                return list(dict.fromkeys(ids))
            after = page[-1]["created_utc"]

    def _search(self, subreddit: str, after: int, before: int) -> List[dict]:
        params = {
            "subreddit": subreddit,
            "after": after,
            "before": before,
            "limit": PAGE_SIZE,
            "sort": "asc",
            "fields": "id,created_utc",
        }
        url = f"{self.base_url}/api/posts/search"
        for attempt in range(MAX_TRIES):
            resp = self.session.get(url, params=params, timeout=TIMEOUT)
            if resp.status_code not in RETRY_STATUS:
                break
            if attempt < MAX_TRIES - 1:
                self.sleep(self._backoff(resp, attempt))
        if resp.status_code != 200:
            raise ArcticShiftError(
                f"search r/{subreddit} after={after}: status={resp.status_code}, body={resp.text[:200]}"
            )
        body = resp.json()
        if body.get("error"):
            raise ArcticShiftError(f"search r/{subreddit}: {body['error']}")
        return body.get("data") or []

    def _backoff(self, resp: requests.Response, attempt: int) -> float:
        reset = resp.headers.get("x-ratelimit-reset")
        if resp.status_code == 429 and reset and reset.isdigit():
            return int(reset) + 1
        return 5 * (attempt + 1)
