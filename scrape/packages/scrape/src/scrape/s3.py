import json
from typing import List, Protocol

from .constants import AWS_BUCKET_COMMENTS
from .models import RedditComment


class S3(Protocol):
    def put_object(self, bucket: str, key: str, body: bytes, content_type: str) -> None:
        """Upload file data. The caller builds the public CloudFront URL."""
        ...


def upload_comments(s3: S3, id: int, comments: List[RedditComment]) -> None:
    body = json.dumps([c.__dict__ for c in comments]).encode("UTF-8")
    s3.put_object(AWS_BUCKET_COMMENTS, f"{id}.json", body, "application/json")
