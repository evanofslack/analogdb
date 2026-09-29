"""Pick a stratified sample of posts, with their text and OP comments.

uv run python -m eval.metadata.sample [--seed 2026]
"""

import argparse
import json
import os
from datetime import datetime, timezone
from typing import Callable, Dict, Iterable, List, Optional, Set

from scrape.constants import AWS_BUCKET_COMMENTS
from scrape.normalize import normalize_key, normalize_tokens

from .common import (
    CATALOG_PATH,
    FIELDS,
    SAMPLE_PATH,
    api_get,
    fetch_catalog,
    load_env,
    write_json,
)

PRE_2025 = int(datetime(2025, 1, 1, tzinfo=timezone.utc).timestamp())

QUOTAS = {
    "negative": 40,
    "not_in_catalog": 40,
    "ids_37k": 30,
    "ids_38k_meta": 90,
    "pre_2025": 100,
}
FEWSHOT = 15
MAX_SCAN = 15000
PAGE_SIZE = 200
COMMENT_CHARS = 1500

# Makes that also sell film, so "Kodak 400" says nothing about the camera
DUAL_MAKES = {"kodak", "fujifilm", "fuji", "polaroid", "rollei"}

# Camera makes people post with that the catalog doesn't have
EXTRA_CAMERA_MAKES = {
    "agfa",
    "ansco",
    "argus",
    "fed",
    "horizon",
    "lomo",
    "lomography",
    "lubitel",
    "nikkormat",
    "rolleiflex",
    "rolleicord",
    "smena",
    "widelux",
    "zenza",
}

EXTRA_BRANDS = EXTRA_CAMERA_MAKES | {
    "cinestill",
    "fomapan",
    "fuji",
    "kodacolor",
    "superia",
    "svema",
    "tmax",
    "trix",
    "velvia",
}

GENERIC_WORDS = {
    "and",
    "blues",
    "camera",
    "china",
    "film",
    "hunter",
    "inc",
    "industries",
    "japan",
    "labs",
    "lucky",
    "oldschool",
    "optik",
    "original",
    "photography",
    "project",
    "psychedelic",
    "yes",
}

FILM_NAME_STOPWORDS = {
    "aero",
    "black",
    "classic",
    "color",
    "colour",
    "double",
    "gold",
    "max",
    "night",
    "pro",
    "silver",
    "super",
    "ultra",
    "white",
}


def _mixed(key: str) -> bool:
    return any(c.isdigit() for c in key) and any(c.isalpha() for c in key)


def brand_words(catalog: Dict) -> Set[str]:
    """Whole words that name a make or a film."""
    words = set(EXTRA_BRANDS)
    makes = [c["make"] for c in catalog["cameras"]] + [
        f["make"] for f in catalog["films"]
    ]
    for make in makes:
        words |= {t for t in normalize_tokens(make) if len(t) >= 3} - GENERIC_WORDS
    for f in catalog["films"]:
        first = normalize_key(f["type"].split()[0])
        if len(first) >= 4 and first not in FILM_NAME_STOPWORDS:
            words.add(first)
    return words


def model_keys(catalog: Dict) -> Set[str]:
    """Letter and digit names (rb67, ae1, hp5) found anywhere in the squashed title."""
    keys = {normalize_key(c["model"]) for c in catalog["cameras"]}
    keys |= {normalize_key(f["type"].split()[0]) for f in catalog["films"]}
    return {k for k in keys if len(k) >= 3 and _mixed(k)}


def camera_models_by_word(catalog: Dict) -> Dict[str, Set[str]]:
    """First word of each camera make → normalized models of that make."""
    models: Dict[str, Set[str]] = {}
    for c in catalog["cameras"]:
        tokens = normalize_tokens(c["make"])
        if not tokens or tokens[0] in DUAL_MAKES:
            continue
        models.setdefault(tokens[0], set()).add(normalize_key(c["model"]))
    return models


def is_negative(post: Dict, brands: Set[str], keys: Set[str]) -> bool:
    title = post["title"]
    if "[" in title:
        return False
    tokens = normalize_tokens(title)
    words = set(tokens) | {a + b for a, b in zip(tokens, tokens[1:])}
    words |= {t.removesuffix("s") for t in words} | {
        t.rstrip("0123456789") for t in words
    }
    if words & brands:
        return False
    squashed = normalize_key(title)
    return not any(k in squashed for k in keys)


def names_uncatalogued_camera(post: Dict, models: Dict[str, Set[str]]) -> bool:
    """A camera make followed by a model-like word that isn't a catalog model."""
    tokens = normalize_tokens(post["title"])
    for i, token in enumerate(tokens[:-1]):
        if token not in models and token not in EXTRA_CAMERA_MAKES:
            continue
        following = tokens[i + 1 : i + 4]
        if not any(c.isdigit() for c in following[0]):
            continue
        if token in EXTRA_CAMERA_MAKES:
            return True
        joined = ["".join(following[: n + 1]) for n in range(len(following))]
        if not any(j in models[token] for j in joined):
            return True
    return False


def has_metadata(post: Dict) -> bool:
    return any(post.get(f) for f in FIELDS)


def stratum_rules(catalog: Dict) -> Dict[str, Callable[[Dict], bool]]:
    brands = brand_words(catalog)
    keys = model_keys(catalog)
    models = camera_models_by_word(catalog)
    return {
        "negative": lambda p: is_negative(p, brands, keys),
        "not_in_catalog": lambda p: names_uncatalogued_camera(p, models),
        "ids_37k": lambda p: 37000 <= p["id"] < 38000,
        "ids_38k_meta": lambda p: p["id"] >= 38000 and has_metadata(p),
        "pre_2025": lambda p: p["timestamp"] < PRE_2025,
    }


def assign_strata(
    posts: Iterable[Dict],
    rules: Dict[str, Callable[[Dict], bool]],
    quotas: Dict[str, int],
    fewshot: int,
) -> List[Dict]:
    """Give each post the first stratum (in quota order) that fits and has room.
    Posts that fit none fill the few-shot set, which is kept out of scoring."""
    counts = {name: 0 for name in quotas}
    picked: List[Dict] = []
    fewshot_count = 0
    for post in posts:
        if all(counts[n] >= quotas[n] for n in quotas) and fewshot_count >= fewshot:
            break
        stratum = next(
            (n for n in quotas if counts[n] < quotas[n] and rules[n](post)), None
        )
        if stratum is not None:
            counts[stratum] += 1
            picked.append({**post, "stratum": stratum, "set": "test"})
        elif fewshot_count < fewshot and any(rules[n](post) for n in rules):
            fewshot_count += 1
            picked.append({**post, "stratum": "fewshot", "set": "fewshot"})
    return picked


def random_posts(seed: int, limit: int) -> Iterable[Dict]:
    params = {"sort": "random", "seed": seed, "page_size": PAGE_SIZE}
    seen = 0
    while seen < limit:
        page = api_get("/posts", params)
        for post in page["posts"]:
            seen += 1
            yield post
        cursor = page["meta"].get("next_cursor")
        if not cursor:
            return
        params = {**params, "cursor": cursor}


def op_comments(comments: List[Dict], author: str) -> List[str]:
    """The post author's own comments, oldest first, capped in total length."""
    author = author.removeprefix("u/")
    own = sorted(
        (c for c in comments if c.get("author", "").removeprefix("u/") == author),
        key=lambda c: c.get("time", 0),
    )
    bodies: List[str] = []
    total = 0
    for c in own:
        body = " ".join(c["body"].split())
        if not body or body in ("[deleted]", "[removed]"):
            continue
        body = body[: COMMENT_CHARS - total]
        bodies.append(body)
        total += len(body)
        if total >= COMMENT_CHARS:
            break
    return bodies


class CommentSource:
    """Reads the S3 comments cache, falling back to Reddit. Never writes."""

    def __init__(self):
        import boto3

        self.s3 = boto3.client("s3")
        self.reddit = None

    def get(self, post: Dict) -> tuple[List[Dict], str]:
        try:
            obj = self.s3.get_object(
                Bucket=AWS_BUCKET_COMMENTS, Key=f"{post['id']}.json"
            )
            return json.loads(obj["Body"].read()), "s3"
        except self.s3.exceptions.NoSuchKey:
            pass
        try:
            comments = self._reddit().scrape_comments(post["permalink"])
            return [c.__dict__ for c in comments], "reddit"
        except Exception as e:
            print(f"  comments failed for {post['id']}: {e}")
            return [], "none"

    def _reddit(self):
        if self.reddit is None:
            import praw
            from scrape.reddit import RedditScraper

            self.reddit = RedditScraper(
                praw.Reddit(
                    client_id=os.environ["REDDIT_CLIENT_ID"],
                    client_secret=os.environ["REDDIT_CLIENT_SECRET"],
                    user_agent=os.environ["REDDIT_USER_AGENT"],
                )
            )
        return self.reddit


def image_url(post: Dict) -> Optional[str]:
    images = post.get("images") or []
    low = next((i for i in images if i["resolution"] == "low"), None)
    return (low or images[0])["url"] if images else None


def to_sample(post: Dict, comments: List[str], source: str) -> Dict:
    return {
        "id": post["id"],
        "set": post["set"],
        "stratum": post["stratum"],
        "title": post["title"],
        "description": post.get("description"),
        "op_comments": comments,
        "comments_source": source,
        "author": post["author"],
        "permalink": post["permalink"],
        "time": post["timestamp"],
        "image": image_url(post),
        "stored": {f: post.get(f) for f in FIELDS},
    }


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--seed", type=int, default=2026)
    args = parser.parse_args()
    load_env()

    catalog = fetch_catalog()
    write_json(CATALOG_PATH, catalog)
    print(f"catalog: {len(catalog['cameras'])} cameras, {len(catalog['films'])} films")

    rules = stratum_rules(catalog)
    picked = assign_strata(random_posts(args.seed, MAX_SCAN), rules, QUOTAS, FEWSHOT)
    counts: Dict[str, int] = {}
    for p in picked:
        counts[p["stratum"]] = counts.get(p["stratum"], 0) + 1
    print(f"picked {len(picked)}: {counts}")

    source = CommentSource()
    samples = []
    for i, post in enumerate(picked, 1):
        comments, where = source.get(post)
        samples.append(to_sample(post, op_comments(comments, post["author"]), where))
        if i % 50 == 0:
            print(f"  comments {i}/{len(picked)}")

    samples.sort(key=lambda s: (s["set"] != "test", s["stratum"], s["id"]))
    sources: Dict[str, int] = {}
    for s in samples:
        sources[s["comments_source"]] = sources.get(s["comments_source"], 0) + 1
    print(f"comments from: {sources}")

    write_json(
        SAMPLE_PATH,
        {
            "seed": args.seed,
            "created": int(datetime.now(timezone.utc).timestamp()),
            "quotas": QUOTAS,
            "fewshot": FEWSHOT,
            "posts": samples,
        },
    )
    print(f"wrote {SAMPLE_PATH}")


if __name__ == "__main__":
    main()
