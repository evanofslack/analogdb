import json
import math
import os
import string
from collections import Counter
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
            if c is None:
                logger.debug("Comment is None, skipping")
                continue

            # deleted account's comments have text but no author
            if c.author.name is None:
                author = "deleted"
            else:
                author = c.author.name

            comment = RedditComment(
                body=c.body,
                score=c.score,
                author=f"u/{author}",
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

    if os.path.exists(filepath):
        print("encountered existing downloaded file, skipping")
        return

    if not os.path.exists(os.path.dirname(filepath)):
        os.makedirs(os.path.dirname(filepath))

    comments = get_comments(reddit=reddit, url=post.permalink)
    with open(filepath, "w") as file:
        json.dump([comment.__dict__ for comment in comments], file)


def read_comments_from_json(filepath: str) -> List[RedditComment]:
    logger.debug(f"reading comments from path: {filepath}")
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
    nlp = spacy.load("en_core_web_lg")

    keywords = []
    pos_tag = ["PROPN", "ADJ", "NOUN"]
    doc = nlp(text.lower())
    printable = set(string.printable)
    union_blacklist = {"http", "www", ".com", "imgur", "wikapedia", "u/", "r/"}
    punctuation = r"""!"#$%&'()*,/:;<=>?@[\]^_`{|}~"""

    for token in doc:
        # no stop words
        if token.text in nlp.Defaults.stop_words:
            continue
        # no single charecters
        if len(token.text) < 2:
            continue
        # no punctuation
        if bool(set(token.text) & set(punctuation)):
            continue
        # we dont want any non printable charecters
        if not set(token.text).issubset(printable):
            continue
        # no blacklisted words
        if blacklist and token.text in blacklist:
            continue
        # substring matching blacklist
        if token.text in union_blacklist:
            continue
        if token.pos_ not in pos_tag:
            continue

        keywords.append(token.text)

    return keywords


def rank_keywords(keywords: List[str], weight: int):
    count = Counter(keywords)

    # no weight, no need to iterate
    if weight <= 1:
        return count

    # multiple count * weight
    for word, _ in count.most_common():
        count[word] *= weight

    return count


def write_keywords_to_disk(keywords: List[AnalogKeyword], filepath: str):

    if not os.path.exists(os.path.dirname(filepath)):
        os.makedirs(os.path.dirname(filepath))

    with open(filepath, "a") as file:
        file.write("\n".join(str(kw.word) for kw in keywords))


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

        # we dont want invalid logarithms
        if int(comment.score) <= 1:
            weight = 1
        else:
            weight = int(math.log(int(comment.score), 2) * 100)
        ranked = rank_keywords(keywords=keywords, weight=weight)
        ranked_keywords += ranked

    return ranked_keywords


def title_counter(title: str, post_score: int) -> Counter:
    keywords = extract_keywords(title)

    # we dont want invalid logarithms
    if post_score <= 1:
        weight = 1
    else:
        weight = int(math.log(int(post_score), 10) * 100)
    ranked_title = rank_keywords(keywords=keywords, weight=weight)

    return ranked_title


def post_keywords(
    title: str,
    comments: List[RedditComment],
    post_score: int,
    limit: Optional[int] = None,
    blacklist: Optional[Set[str]] = None,
) -> List[AnalogKeyword]:
    combined = title_counter(title=title, post_score=post_score) + comment_counter(
        comments=comments
    )
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
        title=title,
        comments=comments,
        post_score=post.score,
        limit=limit,
        blacklist=blacklist,
    )
    return keywords
