from dataclasses import dataclass, field
from enum import Enum
from typing import Dict, Generic, List, Set, Type, TypeVar

from dagster import (
    DagsterType,
    TypeCheck,
    make_python_type_usable_as_dagster_type,
)

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

    def successful_ids(self) -> Set[str]:
        """Get IDs of successfully processed items."""
        return {
            item_id
            for item_id, status in self.status.items()
            if status == Status.SUCCESS
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


def create_result_type_check(expected_inner_type: Type | None = None):
    """Factory function to create type check functions for specific Result types"""

    def type_check_fn(context, value):
        if not isinstance(value, Result):
            return TypeCheck(
                success=False,
                description=f"Expected Result, got {type(value).__name__}",
            )

        if not isinstance(value.data, dict):
            return TypeCheck(
                success=False, description="Result.data must be a dictionary"
            )

        if not isinstance(value.status, dict):
            return TypeCheck(
                success=False, description="Result.status must be a dictionary"
            )

        if not isinstance(value.errors, dict):
            return TypeCheck(
                success=False, description="Result.errors must be a dictionary"
            )

        for status_val in value.status.values():
            if not isinstance(status_val, Status):
                return TypeCheck(
                    success=False,
                    description=f"All status values must be Status enum, found {type(status_val)}",
                )

        # validate inner type if specified
        if expected_inner_type and value.data:
            sample_value = next(iter(value.data.values()))
            if not isinstance(sample_value, expected_inner_type):
                return TypeCheck(
                    success=False,
                    description=f"Expected inner type {expected_inner_type.__name__}, got {type(sample_value).__name__}",
                )

        return TypeCheck(success=True)

    return type_check_fn


ResultDagsterType = DagsterType(
    type_check_fn=create_result_type_check(),
    name="Result",
    description="Generic result container",
    typing_type=Result,
)

make_python_type_usable_as_dagster_type(Result, ResultDagsterType)
