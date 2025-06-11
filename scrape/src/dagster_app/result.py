from dataclasses import dataclass, field
from typing import Dict, Generic, TypeVar, Set, List
from enum import Enum
from scrape.models import PhotoMetadata, RedditPost, UploadPost, S3Image

T = TypeVar("T")


class Status(str, Enum):
    SUCCESS = "success"
    FAILED = "failed"


@dataclass(frozen=True)
class Result(Generic[T]):
    """
    Generic container for processing results that maintains ID correlation
    and tracks success/failure status for each item.
    """

    data: Dict[str, T]
    status: Dict[str, Status]
    errors: Dict[str, str] = field(default_factory=dict)

    def successful(self) -> Dict[str, T]:
        """Get only successfully processed items."""
        return {
            item_id: item
            for item_id, item in self.data.items()
            if self.status.get(item_id) == Status.SUCCESS
        }

    def successful_count(self) -> int:
        """Get only successfully processed items count."""
        success = [
            id for id, _ in self.data.items() if self.status.get(id) == Status.SUCCESS
        ]
        return len(success)

    def failed(self) -> Dict[str, T]:
        """Get only failed items."""
        return {
            item_id: item
            for item_id, item in self.data.items()
            if self.status.get(item_id) == Status.FAILED
        }

    def failed_count(self) -> int:
        """Get only failed processed items count."""
        failed = [
            id for id, _ in self.data.items() if self.status.get(id) == Status.FAILED
        ]
        return len(failed)

    def filter_by_ids(self, item_ids: Set[str]) -> "Result[T]":
        """Create new result with only specified IDs."""
        return Result(
            data={id: self.data[id] for id in item_ids if id in self.data},
            status={id: self.status[id] for id in item_ids if id in self.status},
            errors={id: self.errors[id] for id in item_ids if id in self.errors},
        )


RedditPosts = Result[RedditPost]
TitleMetadatas = Result[PhotoMetadata]
S3Images = Result[List[S3Image]]
FinalPosts = Result[UploadPost]
