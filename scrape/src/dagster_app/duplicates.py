"""Duplicate posts, found by the image hash stored on each post.

`report_duplicates` lists every post that duplicates an older post, which the
backend works out over the whole archive, and unless dry_run is on files a
duplicate report on each one. It never deletes: a person takes duplicates down
from the admin Reports tab.
"""

from typing import Sequence

import analogdb.models as adb
import dagster as dg

from .resources import AnalogDBResource

POST_URL = "https://analogdb.com/post"


class ReportDuplicatesConfig(dg.Config):
    dry_run: bool = True


def _duplicates_table(dupes: Sequence[adb.DuplicatePost], result: str) -> str:
    if not dupes:
        return "None."
    lines = [
        "| keep | duplicate | distance | author | images | result |",
        "|---|---|---|---|---|---|",
    ]
    for d in dupes:
        lines.append(
            f"| [{d.keep_id}]({POST_URL}/{d.keep_id}) "
            f"| [{d.duplicate_id}]({POST_URL}/{d.duplicate_id}) "
            f"| {d.distance} "
            f"| {'same' if d.same_author else 'other'} "
            f"| [keep]({d.keep_url}) [duplicate]({d.duplicate_url}) "
            f"| {result} |"
        )
    return "\n".join(lines)


@dg.asset(group_name="backfill")
def report_duplicates(
    context: dg.AssetExecutionContext,
    config: ReportDuplicatesConfig,
    analogdb: AnalogDBResource,
) -> dg.MaterializeResult:
    """File a duplicate report on every post that duplicates an older post.
    Dry run by default, which only lists them. Posts that already have an open
    duplicate report are skipped, and nothing is deleted."""
    client = analogdb.client()
    if config.dry_run:
        dupes = client.get_duplicates()
        reported = 0
    else:
        dupes, reported = client.report_duplicates()

    metadata = {
        "dry_run": config.dry_run,
        "duplicates": len(dupes),
        "reported": reported,
        "duplicate_ids": str([d.duplicate_id for d in dupes]),
        "pairs": dg.MetadataValue.md(
            _duplicates_table(dupes, "dry run" if config.dry_run else "reported")
        ),
    }
    context.log.info(
        f"Finished report duplicates: dry_run={config.dry_run}, "
        f"duplicates={len(dupes)}, reported={reported}"
    )
    return dg.MaterializeResult(metadata=metadata)
