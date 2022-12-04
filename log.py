import sys

from loguru import logger


def init_logger():
    format = "{time} {level} {message}"
    format_color = "<green>{time}</green> <level> {level} {message}</level>"

    logger.add(sys.stderr, colorize=True, format=format_color, level="INFO")
    logger.add("info.log", format=format, retention="1 week", level="INFO")
    logger.add("error.log", format=format, retention="2 months", level="WARNING")
