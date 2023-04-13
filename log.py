import os
import sys

from loguru import logger
from notifiers.logging import NotificationHandler

from configuration import init_config


def init_logger():

    loglevel = os.environ.get("LOGLEVEL", "INFO").upper()

    format = "{time} {level} {message}"
    format_color = "<green>{time}</green> <level> {level} {message}</level>"

    logger.remove(0)  # remove default handler
    logger.add(sys.stderr, colorize=True, format=format_color, level=loglevel)
    logger.add("info.log", format=format, retention="1 week", level="INFO")
    logger.add("error.log", format=format, retention="2 months", level="WARNING")

    config = init_config()
    slack_params = {"webhook_url": config.slack.url}
    slack_handler = NotificationHandler("slack", defaults=slack_params)
    logger.add(slack_handler, level="WARNING")
