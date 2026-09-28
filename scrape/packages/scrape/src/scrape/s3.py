from typing import Protocol


class S3(Protocol):
    def put_object(self, bucket: str, key: str, body: bytes, content_type: str) -> None:
        """Upload file data. The caller builds the public CloudFront URL."""
        ...
