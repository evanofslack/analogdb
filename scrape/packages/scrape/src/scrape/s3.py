from typing import Protocol


class S3(Protocol):
    def put_object(self, bucket: str, key: str, body: bytes, content_type: str) -> str:
        """Upload file data and return the public URL."""
        ...
