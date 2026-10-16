# /// script
# requires-python = ">=3.11"
# dependencies = ["numpy", "scikit-learn", "psycopg[binary]", "requests", "pillow"]
# ///
"""Fit the visual themes and style prompt vectors for the stats page.

Reads CLIP vectors from a restored Weaviate backup and posts from a restored
Postgres copy, fits k-means centers over the posts in the window and embeds the
style prompts with the CLIP container. Writes to --out:

    seed.json      themes with suggested names, and prompt vectors
    review.md      size, keywords, popularity, B&W and NSFW share per theme
    sheets/*.jpg   a contact sheet of random photos per theme

Name the themes in seed.json from the sheets, set "hidden": true on any that
shouldn't be shown and "role": "border" on the film border theme, then upload
it with `just backend stats-seed`. Nothing here is committed.

Usage:
    uv run scripts/fit_themes.py --out ~/.cache/analogdb/stats
"""

import argparse
import json
import random
from collections import Counter
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from io import BytesIO
from pathlib import Path

import numpy as np
import psycopg
import requests
from PIL import Image
from sklearn.cluster import KMeans

MODEL = "ViT-B-16-laion2b_s34b_b88k"
WINDOW_START = datetime(2022, 1, 1, tzinfo=timezone.utc)
IMAGE_URL = "https://i.analogdb.com/{}"
PAGE_SIZE = 500
MIN_KEYWORD_SHARE = 0.12
SHEET_PHOTOS = 12
SHEET_COLUMNS = 4
SHEET_CELL = 240

PROMPTS = [
    ("base", "Base", "a photo"),
    ("vivid", "Vivid", "a photo with vivid saturated colors"),
    ("teal_orange", "Teal and orange", "a photo with teal and orange colors"),
    ("red_night", "Red night light", "a photo with red halation glow around lights"),
    ("grain", "Grain", "a photo with heavy film grain"),
    ("crisp", "Crisp", "a crisp sharp detailed photo"),
]

POSTS_QUERY = """
    SELECT id, lowurl, COALESCE(nsfw, false), COALESCE(greyscale, false),
        percent_rank() OVER (
            PARTITION BY date_trunc('month', to_timestamp(time) AT TIME ZONE 'UTC')
            ORDER BY score
        )
    FROM pictures
    WHERE time >= %s AND score IS NOT NULL
"""

KEYWORDS_QUERY = """
    SELECT k.post_id, k.word
    FROM keywords k
    JOIN pictures p ON p.id = k.post_id
    WHERE p.time >= %s
"""


def export_vectors(weaviate: str) -> tuple[np.ndarray, np.ndarray]:
    ids, vectors = [], []
    after = None
    while True:
        cursor = f', after: "{after}"' if after else ""
        query = f"{{Get{{PostImage(limit: {PAGE_SIZE}{cursor}){{post_id _additional{{id vector}}}}}}}}"
        resp = requests.post(
            f"{weaviate}/v1/graphql", json={"query": query}, timeout=120
        )
        resp.raise_for_status()
        body = resp.json()
        if body.get("errors"):
            raise RuntimeError(body["errors"])
        objects = body["data"]["Get"]["PostImage"]
        if not objects:
            break
        for obj in objects:
            if obj["post_id"] is not None:
                ids.append(obj["post_id"])
                vectors.append(obj["_additional"]["vector"])
        after = objects[-1]["_additional"]["id"]
        print(f"exported {len(ids)} vectors", end="\r", flush=True)
    print()
    return np.array(ids, dtype=np.int64), np.array(vectors, dtype=np.float32)


def load_vectors(
    weaviate: str, out: Path, refresh: bool
) -> tuple[np.ndarray, np.ndarray]:
    ids_path, vectors_path = out / "ids.npy", out / "vectors.npy"
    if ids_path.exists() and vectors_path.exists() and not refresh:
        return np.load(ids_path), np.load(vectors_path)
    ids, vectors = export_vectors(weaviate)
    np.save(ids_path, ids)
    np.save(vectors_path, vectors)
    return ids, vectors


def normalize(x: np.ndarray) -> np.ndarray:
    return x / np.linalg.norm(x, axis=-1, keepdims=True)


def embed_prompts(clip: str) -> np.ndarray:
    texts = [text for _, _, text in PROMPTS]
    resp = requests.post(
        f"{clip}/vectorize", json={"texts": texts, "images": []}, timeout=120
    )
    resp.raise_for_status()
    return normalize(np.array(resp.json()["textVectors"], dtype=np.float32))


def keyword_lift(
    members: list[int], keywords: dict[int, set[str]], overall: Counter, total: int
) -> list[tuple[str, float, float]]:
    counts = Counter(w for pid in members for w in keywords.get(pid, ()))
    rows = []
    for word, n in counts.items():
        share = n / len(members)
        if share >= MIN_KEYWORD_SHARE:
            rows.append((word, share, share / (overall[word] / total)))
    return sorted(rows, key=lambda r: r[2], reverse=True)


def fetch_thumb(lowurl: str, cache: Path) -> Image.Image | None:
    path = cache / lowurl
    if not path.exists():
        try:
            resp = requests.get(IMAGE_URL.format(lowurl), timeout=30)
            resp.raise_for_status()
        except requests.RequestException as e:
            print(f"skipping {lowurl}: {e}")
            return None
        path.write_bytes(resp.content)
    try:
        return Image.open(BytesIO(path.read_bytes())).convert("RGB")
    except OSError:
        return None


def contact_sheet(lowurls: list[str], cache: Path) -> Image.Image:
    with ThreadPoolExecutor(8) as pool:
        thumbs = [
            t
            for t in pool.map(lambda u: fetch_thumb(u, cache), lowurls)
            if t is not None
        ]
    rows = max(1, -(-len(thumbs) // SHEET_COLUMNS))
    sheet = Image.new("RGB", (SHEET_COLUMNS * SHEET_CELL, rows * SHEET_CELL), "white")
    for i, thumb in enumerate(thumbs):
        thumb.thumbnail((SHEET_CELL - 4, SHEET_CELL - 4))
        x = (i % SHEET_COLUMNS) * SHEET_CELL + (SHEET_CELL - thumb.width) // 2
        y = (i // SHEET_COLUMNS) * SHEET_CELL + (SHEET_CELL - thumb.height) // 2
        sheet.paste(thumb, (x, y))
    return sheet


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    parser.add_argument("--weaviate", default="http://127.0.0.1:58080")
    parser.add_argument("--clip", default="http://127.0.0.1:58081")
    parser.add_argument(
        "--dsn", default="postgres://postgres:postgres@127.0.0.1:55433/analogdb"
    )
    parser.add_argument("--out", type=Path, default=Path("~/.cache/analogdb/stats"))
    parser.add_argument("--k", type=int, default=36)
    parser.add_argument(
        "--refresh", action="store_true", help="export the vectors again"
    )
    args = parser.parse_args()

    out = args.out.expanduser()
    sheets, thumbs = out / "sheets", out / "thumbs"
    for d in (out, sheets, thumbs):
        d.mkdir(parents=True, exist_ok=True)

    ids, vectors = load_vectors(args.weaviate, out, args.refresh)

    start = int(WINDOW_START.timestamp())
    with psycopg.connect(args.dsn) as conn:
        posts = {r[0]: r[1:] for r in conn.execute(POSTS_QUERY, (start,))}
        keywords: dict[int, set[str]] = {}
        for pid, word in conn.execute(KEYWORDS_QUERY, (start,)):
            keywords.setdefault(pid, set()).add(word)

    _, first = np.unique(ids, return_index=True)
    keep = np.array(sorted(i for i in first if int(ids[i]) in posts))
    ids, vectors = ids[keep], normalize(vectors[keep])
    print(f"{len(ids)} posts with vectors in the window")

    kmeans = KMeans(n_clusters=args.k, n_init=4, random_state=0).fit(vectors)
    centers = normalize(kmeans.cluster_centers_.astype(np.float32))
    labels = kmeans.labels_
    order = sorted(range(args.k), key=lambda c: -(labels == c).sum())

    overall = Counter(w for pid in ids.tolist() for w in keywords.get(pid, ()))
    total = len(ids)
    rng = random.Random(0)

    themes, review = [], [f"# Visual themes ({args.k}, {total} posts)\n"]
    for rank, c in enumerate(order):
        key = f"t{rank:02d}"
        members = ids[labels == c].tolist()
        lift = keyword_lift(members, keywords, overall, total)
        name = ", ".join(w for w, _, _ in lift[:3]) or key
        rows = [posts[pid] for pid in members]
        popularity = 100 * np.mean([r[3] for r in rows])
        bw = 100 * np.mean([r[2] for r in rows])
        nsfw = 100 * np.mean([r[1] for r in rows])

        sample = rng.sample(members, min(SHEET_PHOTOS, len(members)))
        contact_sheet([posts[pid][0] for pid in sample if posts[pid][0]], thumbs).save(
            sheets / f"{key}.jpg", quality=85
        )

        themes.append(
            {
                "key": key,
                "name": name,
                "hidden": False,
                "role": None,
                "center": centers[c].tolist(),
            }
        )
        words = ", ".join(f"{w} {100 * s:.0f}% {lft:.1f}×" for w, s, lft in lift[:8])
        review.append(
            f"## {key}: {name}\n\n"
            f"{len(members)} posts · popularity {popularity:.0f} · B&W {bw:.0f}% · NSFW {nsfw:.0f}%\n\n"
            f"Keywords: {words or 'none'}\n\n"
            f"![{key}](sheets/{key}.jpg)\n"
        )
        print(f"{key} {len(members):5d} pop {popularity:4.0f} bw {bw:3.0f}% {name}")

    prompt_vectors = embed_prompts(args.clip)
    prompts = [
        {"key": key, "label": label, "text": text, "vector": prompt_vectors[i].tolist()}
        for i, (key, label, text) in enumerate(PROMPTS)
    ]

    seed = {
        "version": datetime.now(timezone.utc).isoformat(timespec="seconds"),
        "model": MODEL,
        "themes": themes,
        "prompts": prompts,
    }
    (out / "seed.json").write_text(json.dumps(seed))
    (out / "review.md").write_text("\n".join(review))
    print(f"wrote {out / 'seed.json'} and {out / 'review.md'}")


if __name__ == "__main__":
    main()
