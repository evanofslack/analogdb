"""The stats page snapshot, rebuilt by the backend from everything stored.

`rebuild_stats` runs after the nightly jobs so new posts, scores, captions
and vectors are in it. The backend runs one build at a time, so a build that
is already running is not an error.
"""

import dagster as dg
from analogdb_generated.exceptions import ApiException

from .resources import AnalogDBResource


@dg.asset(group_name="stats")
def rebuild_stats(
    context: dg.AssetExecutionContext,
    analogdb: AnalogDBResource,
) -> dg.MaterializeResult:
    """Build a new stats snapshot, served from GET /v1/stats."""
    try:
        build = analogdb.client().rebuild_stats()
    except ApiException as e:
        if e.status == 409:
            context.log.info("Skipped stats rebuild, a build is already running")
            return dg.MaterializeResult(metadata={"skipped": True})
        raise

    context.log.info(
        f"Finished stats rebuild: snapshot={build.id}, build_ms={build.build_ms}, "
        f"films={build.films}, themes={build.themes}"
    )
    return dg.MaterializeResult(
        metadata={
            "skipped": False,
            "snapshot_id": build.id,
            "build_ms": build.build_ms,
            "films": build.films,
            "themes": build.themes,
        }
    )
