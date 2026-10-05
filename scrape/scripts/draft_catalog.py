"""Draft new camera and film catalog entries from the unmatched mentions.

1. candidates: rank the stored extractions' unmatched mentions and write a
   triage file with one item per key, action "todo".
2. Edit drafts/triage.json by hand: set each item's action to "new" (with make,
   model or type, speed and color_type for films), "alias" (with "target", the
   entry's model or type, and "aliases") or "skip". "source" pins a camera-wiki
   page title, or "wikipedia:Title".
3. draft: look up a source for each new item and draft its description with an
   LLM from it. Items without a source aren't drafted: pin a page and rerun.
   Writes drafts/drafts.json and drafts/review.md. Rerun to retry failures,
   drafts already written are kept unless --force.
4. Read review.md, fix descriptions in drafts.json, set "rejected": true to drop one.
5. apply: add the drafts and aliases to data/cameras.json and data/films.json.

Usage:
    uv run python scripts/draft_catalog.py candidates --min-posts 5
    uv run python scripts/draft_catalog.py draft
    uv run python scripts/draft_catalog.py apply

Requires ANALOGDB_ENDPOINT, ANALOGDB_USERNAME, ANALOGDB_PASSWORD for candidates
and OPENROUTER_API_KEY (and optionally OPENROUTER_BASE_URL) for draft.
Everything is written to drafts/, which is not committed.
"""

import argparse
import json
import os
import time
from concurrent.futures import ThreadPoolExecutor
from dataclasses import asdict
from pathlib import Path
from typing import Any, Dict

import requests
from analogdb.client import Client
from openai import OpenAI
from scrape.catalog_drafts import (
    apply_drafts,
    candidates,
    draft_prompt,
    find_source,
    name_field,
    parse_draft,
    pick_examples,
    review_markdown,
)

from dagster_app.catalog import validate_aliases

ROOT = Path(__file__).resolve().parents[1]
DATA = ROOT / "data"
DRAFTS = ROOT / "drafts"
USER_AGENT = "analogdb-catalog/0.1 (https://analogdb.com)"
DEFAULT_MODEL = "google/gemini-2.5-flash"


def read_json(path: Path) -> Any:
    with open(path) as f:
        return json.load(f)


def write_json(path: Path, data: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w") as f:
        f.write(json.dumps(data, indent=2, ensure_ascii=False) + "\n")


def item_key(item: Dict[str, Any]) -> tuple:
    return (item["kind"], item["key"])


def run_candidates(args) -> None:
    client = Client(
        base_url=os.environ["ANALOGDB_ENDPOINT"],
        username=os.environ["ANALOGDB_USERNAME"],
        password=os.environ["ANALOGDB_PASSWORD"],
    )
    extractions = [e.to_dict() for e in client.iter_extractions(has_unmatched=True)]
    found = candidates(
        extractions,
        read_json(DATA / "cameras.json"),
        read_json(DATA / "films.json"),
        args.min_posts,
    )
    write_json(DRAFTS / "candidates.json", [asdict(c) for c in found])

    triage_path = DRAFTS / "triage.json"
    triage = read_json(triage_path) if triage_path.exists() else []
    known = {item_key(i) for i in triage}
    for c in found:
        if (c.kind, c.key) in known:
            continue
        item = {
            "kind": c.kind,
            "key": c.key,
            "posts": c.posts,
            "spellings": c.spellings,
            "nearest": c.nearest,
            "action": "todo",
            "make": c.make,
            name_field(c.kind): c.name,
        }
        if c.kind == "film":
            item.update(speed=c.speed, color_type=None)
        triage.append(item)
    write_json(triage_path, triage)
    print(f"{len(found)} candidates, {len(triage)} triage items in {triage_path}")


def fetch_json(url: str, params: Dict[str, Any]) -> Dict[str, Any]:
    for attempt in range(4):
        resp = requests.get(
            url, params=params, headers={"User-Agent": USER_AGENT}, timeout=30
        )
        if resp.status_code in (429, 500, 502, 503) and attempt < 3:
            time.sleep(2 ** (attempt + 1))
            continue
        resp.raise_for_status()
        time.sleep(0.5)
        return resp.json()
    raise RuntimeError(f"failed to fetch {url}")


def complete(
    ai: OpenAI, model: str, system: str, user: str, kind: str
) -> Dict[str, str]:
    error = None
    for _ in range(2):
        messages = [
            {"role": "system", "content": system},
            {"role": "user", "content": user},
        ]
        if error:
            messages.append(
                {"role": "user", "content": f"That reply was invalid: {error}"}
            )
        resp = ai.chat.completions.create(
            model=model,
            messages=messages,
            response_format={"type": "json_object"},
            temperature=0.3,
        )
        try:
            return parse_draft(resp.choices[0].message.content or "", kind)
        except ValueError as e:
            error = str(e)
    raise ValueError(error)


def run_draft(args) -> None:
    triage = read_json(DRAFTS / "triage.json")
    todo = [i for i in triage if i.get("action") == "todo"]
    if todo:
        print(f"Warning: {len(todo)} triage items still have action todo")
    drafts_path = DRAFTS / "drafts.json"
    previous = {}
    if drafts_path.exists() and not args.force:
        previous = {item_key(i): i for i in read_json(drafts_path)}

    entries = {
        "camera": read_json(DATA / "cameras.json"),
        "film": read_json(DATA / "films.json"),
    }
    items = []
    pending = []
    for item in triage:
        if item.get("action") not in ("new", "alias"):
            continue
        done = previous.get(item_key(item))
        redo = args.redo and item["key"] in args.redo
        if item["action"] == "new" and done and done.get("description") and not redo:
            items.append({**item, **{k: v for k, v in done.items() if k not in item}})
            continue
        items.append(dict(item))
        if item["action"] == "new":
            pending.append(items[-1])
    if args.limit is not None:
        pending = pending[: args.limit]

    print(f"Looking up sources for {len(pending)} entries")
    sources = {}
    for item in pending:
        try:
            sources[id(item)] = find_source(item, fetch_json)
        except Exception as e:
            sources[id(item)] = None
            item["error"] = f"source: {e}"
        source = sources[id(item)]
        item["source_title"] = source.title if source else None
        item["source_url"] = source.url if source else None

    ai = OpenAI(
        base_url=os.environ.get("OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"),
        api_key=os.environ["OPENROUTER_API_KEY"],
    )

    def draft_one(item):
        kind = item["kind"]
        if sources[id(item)] is None:
            # Not drafted without a source: pin one in the triage file and draft again
            item.setdefault("error", "no source, pin one with source")
            return
        examples = pick_examples(item, entries[kind])
        system, user = draft_prompt(item, sources[id(item)], examples)
        try:
            out = complete(ai, args.model, system, user, kind)
        except Exception as e:
            item["error"] = f"draft: {e}"
            return
        item["description"] = out["description"]
        if kind == "film" and not item.get("color_type"):
            item["color_type"] = out["color_type"]
        item.pop("error", None)

    print(f"Drafting {len(pending)} descriptions with {args.model}")
    with ThreadPoolExecutor(max_workers=args.workers) as pool:
        list(pool.map(draft_one, pending))

    write_json(drafts_path, items)
    (DRAFTS / "review.md").write_text(review_markdown(items))
    failed = [i for i in items if i.get("action") == "new" and not i.get("description")]
    unsourced = [
        i for i in items if i.get("action") == "new" and not i.get("source_url")
    ]
    print(
        f"Wrote {drafts_path}: {len(items)} items, {len(failed)} failed, "
        f"{len(unsourced)} without a source"
    )


def run_apply(args) -> None:
    items = [i for i in read_json(DRAFTS / "drafts.json") if not i.get("rejected")]
    missing = [
        i for i in items if i.get("action") == "new" and not i.get("description")
    ]
    if missing:
        names = ", ".join(f"{i['make']} {i[name_field(i['kind'])]}" for i in missing)
        raise SystemExit(f"No description for: {names}. Draft again or reject them.")
    cameras, films = apply_drafts(
        read_json(DATA / "cameras.json"), read_json(DATA / "films.json"), items
    )
    problems = validate_aliases(cameras, films)
    if problems:
        raise SystemExit("Alias problems:\n" + "\n".join(problems))
    write_json(DATA / "cameras.json", cameras)
    write_json(DATA / "films.json", films)
    new = sum(1 for i in items if i.get("action") == "new")
    aliases = sum(1 for i in items if i.get("action") == "alias")
    print(f"Added {new} entries and {aliases} alias items")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__.splitlines()[0])
    sub = parser.add_subparsers(dest="command", required=True)
    p = sub.add_parser("candidates")
    p.add_argument("--min-posts", type=int, default=5)
    p.set_defaults(run=run_candidates)
    p = sub.add_parser("draft")
    p.add_argument("--model", default=DEFAULT_MODEL)
    p.add_argument("--workers", type=int, default=4)
    p.add_argument("--force", action="store_true")
    p.add_argument("--limit", type=int, help="draft only the first n new items")
    p.add_argument("--redo", nargs="*", help="triage keys to draft again")
    p.set_defaults(run=run_draft)
    p = sub.add_parser("apply")
    p.set_defaults(run=run_apply)
    args = parser.parse_args()
    args.run(args)


if __name__ == "__main__":
    main()
