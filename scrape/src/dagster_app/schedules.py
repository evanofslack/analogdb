import dagster as dg

from .jobs import scrape_job

scrape_analog_schedule = dg.ScheduleDefinition(
    job=scrape_job, cron_schedule="0 8 * * *"
)
