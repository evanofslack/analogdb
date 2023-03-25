import json
import os
from collections import Counter
from string import punctuation
from typing import List, Optional, Set

import praw
import spacy
from loguru import logger

from constants import REDDIT_URL
from models import AnalogDisplayPost, AnalogKeyword, RedditComment


def get_comments(reddit: praw.Reddit, url: str) -> List[RedditComment]:
    submission = reddit.submission(url=url)

    comments: List[RedditComment] = []

    # follow all comment trees
    submission.comments.replace_more(limit=None)

    # iterate over posts comments and convert to native type
    for c in submission.comments.list():
        try:
            comment = RedditComment(
                body=c.body,
                score=c.score,
                author=f"u/{c.author.name}",
                time=int(c.created_utc),
                permalink=f"{REDDIT_URL}{c.permalink}",
            )
            comments.append(comment)
        except Exception as e:
            logger.info(f"Error extracting comment for {url}, error: {e}")
            continue

    return comments


def write_comments_to_json(reddit: praw.Reddit, post: AnalogDisplayPost):

    filepath = f"comments/{post.id}.json"

    if not os.path.exists(os.path.dirname(filepath)):
        os.makedirs(os.path.dirname(filepath))

    comments = get_comments(reddit=reddit, url=post.permalink)
    with open(filepath, "w") as file:
        json.dump([comment.__dict__ for comment in comments], file)


def read_comments_from_json(filepath: str) -> List[RedditComment]:
    with open(filepath, "r") as f:
        comments_json = json.load(f)

    comments = [
        RedditComment(
            comment["body"],
            comment["score"],
            comment["author"],
            comment["time"],
            comment["permalink"],
        )
        for comment in comments_json
    ]
    return comments


def extract_keywords(text: str, blacklist: Optional[Set[str]] = None) -> List[str]:
    # load model
    nlp = spacy.load("en_core_web_sm")

    keywords = []
    pos_tag = ["PROPN", "ADJ", "NOUN"]
    doc = nlp(text.lower())

    for token in doc:
        if token.text in nlp.Defaults.stop_words or token.text in punctuation:
            continue
        if punctuation in token.text:
            continue
        if blacklist and token.text in blacklist:
            continue
        if token.pos_ in pos_tag:
            keywords.append(token.text)

    return keywords


def rank_keywords(keywords: List[str], weight: int):
    count = Counter(keywords)

    # no weight, no need to iterate
    if weight == 1:
        return count

    # multiple count * weight
    for word, _ in count.most_common():
        count[word] *= weight

    return count


def counter_to_keywords(
    counter: Counter, limit: Optional[int] = None
) -> List[AnalogKeyword]:

    keywords: List[AnalogKeyword] = []

    total = counter.total()
    for word, score in counter.most_common(n=limit):
        keyword = AnalogKeyword(word=word, weight=score / total)
        keywords.append(keyword)

    return keywords


def remove_from_counter(counter: Counter, blacklist: Set[str]) -> Counter:
    remove = [word for word in counter.keys() if word in blacklist]
    for word in remove:
        del counter[word]
    return counter


def comment_counter(comments: List[RedditComment]) -> Counter:
    ranked_keywords: Counter = Counter()

    for comment in comments:
        keywords = extract_keywords(comment.body)
        ranked = rank_keywords(keywords=keywords, weight=1)
        ranked_keywords += ranked

    return ranked_keywords


def title_counter(title: str) -> Counter:
    title_keywords = extract_keywords(title)
    # title_weight = int(post.score / (len(comments) * len(comments)))
    title_weight = 1
    title_ranked = rank_keywords(keywords=title_keywords, weight=title_weight)

    return title_ranked


def post_keywords(
    title: str,
    comments: List[RedditComment],
    limit: Optional[int] = None,
    blacklist: Optional[Set[str]] = None,
) -> List[AnalogKeyword]:
    combined = title_counter(title=title) + comment_counter(comments=comments)
    if blacklist is not None:
        combined = remove_from_counter(counter=combined, blacklist=blacklist)
    keywords = counter_to_keywords(counter=combined, limit=limit)
    return keywords


def post_keywords_from_disk(
    post: AnalogDisplayPost,
    limit: Optional[int] = None,
    blacklist: Optional[Set[str]] = None,
) -> List[AnalogKeyword]:

    title = post.title
    filepath = f"comments/{post.id}.json"
    comments = read_comments_from_json(filepath=filepath)

    keywords = post_keywords(
        title=title, comments=comments, limit=limit, blacklist=blacklist
    )
    return keywords
