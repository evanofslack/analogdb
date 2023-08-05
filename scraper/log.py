import sys

from loguru import logger
from notifiers.logging import NotificationHandler

from configuration import init_config


def init_logger():

    config = init_config()

    format = "{time} | {level} | {message}"

    serialize = False
    diagnose = True
    colorize = True

    if config.app.env == "prod" or config.app.env == "production":
        serialize = True
        diagnose = False
        colorize = False

    logger.remove(0)  # remove default handler
    logger.add(
        sys.stderr,
        colorize=colorize,
        format=format,
        level=config.app.log_level,
        serialize=serialize,
        diagnose=diagnose,
        backtrace=True,
    )

    logger.info(
        f"created new logger with app_env={config.app.env}, level={config.app.log_level}"
    )

    if config.slack.url != "":
        logger.info("adding slack log notifier")
        slack_params = {"webhook_url": config.slack.url}
        slack_handler = NotificationHandler("slack", defaults=slack_params)
        logger.add(slack_handler, level="WARNING")
