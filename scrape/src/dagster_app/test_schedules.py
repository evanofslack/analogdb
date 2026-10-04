from datetime import UTC, datetime

import dagster as dg

from .backfill import ReencodeVectorsConfig
from .definitions import defs
from .resources import TaggerResource
from .schedules import (
    encode_missing_vectors_schedule,
    update_post_comments_schedule,
    update_post_scores_schedule,
)

RETRY_ASSETS = {"reddit_posts", "post_images", "upload_posts"}


def evaluate(schedule: dg.ScheduleDefinition) -> dg.RunRequest:
    context = dg.build_schedule_context(
        scheduled_execution_time=datetime(2026, 9, 27, 1, tzinfo=UTC)
    )
    request = schedule(context)
    assert isinstance(request, dg.RunRequest)
    return request


def test_update_post_scores_schedule():
    request = evaluate(update_post_scores_schedule)
    assert request.partition_key == "2026-09-25"
    assert request.run_key == "scores-2026-09-25"
    assert request.tags["partition"] == "2026-09-25"


def test_update_post_comments_schedule():
    request = evaluate(update_post_comments_schedule)
    assert request.partition_key == "2026-09-25"
    assert request.run_key == "comments-2026-09-25"
    assert request.tags["partition"] == "2026-09-25"


def test_encode_missing_vectors_schedule():
    schedule = encode_missing_vectors_schedule
    assert schedule.cron_schedule == "0 3 * * *"
    assert schedule.job_name == "reencode_vectors"
    context = dg.build_schedule_context(
        scheduled_execution_time=datetime(2026, 9, 27, 3, tzinfo=UTC)
    )
    (request,) = schedule.evaluate_tick(context).run_requests
    assert request.run_config == {
        "ops": {
            "reencode_post_vectors": {
                "config": {"missing_only": True, "fail_on_error": False}
            }
        }
    }
    config = request.run_config["ops"]["reencode_post_vectors"]["config"]
    assert ReencodeVectorsConfig(**config).fail_on_error is False
    assert (
        "reencode_post_vectors"
        in defs.resolve_job_def("reencode_vectors").graph.node_dict
    )


def test_schedules_run_in_utc():
    for schedule in defs.schedules:
        assert schedule.execution_timezone == "UTC"


def test_definitions_load():
    dg.Definitions.validate_loadable(defs)


def test_tagger_defaults_to_flash():
    assert TaggerResource().openai_model == "google/gemini-2.5-flash"


def test_scrape_job_tags_posts():
    job = defs.resolve_job_def("scrape_and_upload")
    assert "post_tags" in job.graph.node_dict
    assert "keywords" not in job.graph.node_dict


def test_day_two_job_keeps_comments_and_metadata():
    job = defs.resolve_job_def("update_post_comments")
    assert set(job.graph.node_dict) == {
        "analogdb_posts",
        "updated_reddit_comments",
        "reddit_comments_to_s3",
        "patch_post_metadata",
    }


def test_retry_policies():
    retried = set()
    for asset in defs.assets:
        policy = asset.op.retry_policy
        if policy is not None:
            assert policy.max_retries == 2
            assert policy.delay == 60
            assert policy.backoff == dg.Backoff.EXPONENTIAL
            retried.add(asset.key.to_user_string())
    assert retried == RETRY_ASSETS
