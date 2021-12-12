import praw
import requests
from PIL import Image

reddit = praw.Reddit("bot_1")

for submission in reddit.subreddit("analog").top(limit=5):
    if not submission.is_self:
        pic = requests.get(submission.url, stream=True)
        img = Image.open(pic.raw)
        img.show()
