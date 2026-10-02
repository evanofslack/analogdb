from datetime import timedelta

import dagster as dg

from .jobs import patch_comments_job, patch_scores_job, scrape_job

scrape_analog_schedule = dg.ScheduleDefinition(
    job=scrape_job,
    cron_schedule="0 0 * * *",
    execution_timezone="UTC",
    name="scrape_analog_schedule",
    description="Daily scrape of posts",
)


def two_days_ago(context: dg.ScheduleEvaluationContext) -> str:
    return (context.scheduled_execution_time - timedelta(days=2)).strftime("%Y-%m-%d")


@dg.schedule(
    job=patch_scores_job,
    cron_schedule="0 1 * * *",
    execution_timezone="UTC",
    name="update_post_scores_schedule",
    description="Daily update of post scores for two days past partition",
)
def update_post_scores_schedule(context: dg.ScheduleEvaluationContext):
    twodays = two_days_ago(context)

    return dg.RunRequest(
        run_key=f"scores-{twodays}",
        partition_key=twodays,
        tags={"schedule": "daily_post_scores", "partition": twodays},
    )


@dg.schedule(
    job=patch_comments_job,
    cron_schedule="30 1 * * *",
    execution_timezone="UTC",
    name="update_post_comments_schedule",
    description="Daily comments and metadata update for two days past partition",
)
def update_post_comments_schedule(context: dg.ScheduleEvaluationContext):
    twodays = two_days_ago(context)

    return dg.RunRequest(
        run_key=f"comments-{twodays}",
        partition_key=twodays,
        tags={"schedule": "daily_post_comments", "partition": twodays},
    )
