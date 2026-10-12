"""Duplicate posts, found by the image hash stored on each post.

`backfill_image_hashes` hashes the low res image of every post missing a hash.
New posts get their hash at upload, so the missing list only shrinks and a rerun
picks up where the last one stopped.

`report_duplicates` lists every post that duplicates an older post, which the
backend works out over the whole archive, and unless dry_run is on files a
duplicate report on each one. It never deletes: a person takes duplicates down
from the admin Reports tab.
"""

from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from typing import Dict, List, Optional, Sequence

import analogdb.models as adb
import dagster as dg
import requests
from scrape.constants import IMAGE_TIMEOUT
from scrape.image import image_bytes_hash

from .assets import error_detail
from .backfill import BACKFILL_CHUNK, chunks
from .resources import AnalogDBResource

POST_URL = "https://analogdb.com/post"


class BackfillImageHashesConfig(dg.Config):
    limit: Optional[int] = None
    concurrency: int = 16


@dataclass
class HashStats:
    posts: int = 0
    written: int = 0
    download_failed: int = 0
    write_failed: int = 0

    def metadata(self) -> Dict[str, int]:
        return dict(self.__dict__)


def download_hash(url: str) -> str:
    resp = requests.get(url, timeout=IMAGE_TIMEOUT)
    resp.raise_for_status()
    return image_bytes_hash(resp.content)


@dg.asset(group_name="backfill")
def backfill_image_hashes(
    context: dg.AssetExecutionContext,
    config: BackfillImageHashesConfig,
    analogdb: AnalogDBResource,
) -> dg.MaterializeResult:
    """Hash the low res image of every post missing an image hash."""
    client = analogdb.client()
    missing = client.get_missing_hashes()
    if config.limit is not None:
        missing = missing[: config.limit]
    context.log.info(f"Found {len(missing)} posts missing an image hash")
    stats = HashStats()

    with ThreadPoolExecutor(max_workers=config.concurrency) as pool:
        for chunk in chunks(missing, BACKFILL_CHUNK):
            stats.posts += len(chunk)
            futures = [(p, pool.submit(download_hash, p.url)) for p in chunk]
            hashes: List[adb.ImageHash] = []
            for p, future in futures:
                try:
                    hashes.append(adb.ImageHash(id=p.id, image_hash=future.result()))
                except Exception as e:
                    stats.download_failed += 1
                    context.log.warning(f"Failed to hash post image, id={p.id}, {e}")
            try:
                stats.written += client.set_image_hashes(hashes)
            except Exception as e:
                stats.write_failed += len(hashes)
                context.log.error(
                    f"Failed to write {len(hashes)} image hashes, {error_detail(e)}"
                )
            context.log.info(
                f"Image hash progress: {stats.posts} of {len(missing)} posts, "
                f"written={stats.written}, download_failed={stats.download_failed}, "
                f"write_failed={stats.write_failed}"
            )

    metadata = stats.metadata()
    context.log.info(f"Finished backfill image hashes: {metadata}")
    if stats.write_failed:
        raise dg.Failure(
            description=f"backfill image hashes: {stats.write_failed} writes failed",
            metadata=metadata,
        )
    return dg.MaterializeResult(metadata=metadata)


class ReportDuplicatesConfig(dg.Config):
    dry_run: bool = True


def _duplicates_table(dupes: Sequence[adb.DuplicatePost], result: str) -> str:
    if not dupes:
        return "None."
    lines = [
        "| keep | duplicate | distance | images | result |",
        "|---|---|---|---|---|",
    ]
    for d in dupes:
        lines.append(
            f"| [{d.keep_id}]({POST_URL}/{d.keep_id}) "
            f"| [{d.duplicate_id}]({POST_URL}/{d.duplicate_id}) "
            f"| {d.distance} "
            f"| [keep]({d.keep_url}) [duplicate]({d.duplicate_url}) "
            f"| {result} |"
        )
    return "\n".join(lines)


@dg.asset(group_name="backfill", deps=[backfill_image_hashes])
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
