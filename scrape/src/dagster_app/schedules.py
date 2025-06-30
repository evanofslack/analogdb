from datetime import datetime, timedelta

import dagster as dg

from .jobs import patch_keywords_job, patch_scores_job, scrape_job

scrape_analog_schedule = dg.ScheduleDefinition(
    job=scrape_job,
    cron_schedule="0 0 * * *",
    name="scrape_analog_schedule",
    description="Daily scrape of posts",
)


@dg.schedule(
    job=patch_scores_job,
    cron_schedule="0 1 * * *",
    name="update_post_scores_schedule",
    description="Daily update of post scores for two days past partition",
)
def update_post_scores_schedule(context: dg.ScheduleEvaluationContext):
    # Get two days ago date for the partition
    twodays = (datetime.now() - timedelta(days=2)).strftime("%Y-%m-%d")

    return dg.RunRequest(
        partition_key=twodays,
        tags={"schedule": "daily_post_scores", "partition": twodays},
    )


@dg.schedule(
    job=patch_keywords_job,
    cron_schedule="0 1 * * *",
    name="update_post_keywords_schedule",
    description="Daily update of post keywords for two days past partition",
)
def update_post_keywords_schedule(context: dg.ScheduleEvaluationContext):
    # Get two days ago date for the partition
    twodays = (datetime.now() - timedelta(days=2)).strftime("%Y-%m-%d")

    return dg.RunRequest(
        partition_key=twodays,
        tags={"schedule": "daily_keywords_scores", "partition": twodays},
    )
