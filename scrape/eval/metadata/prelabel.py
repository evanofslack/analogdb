"""Draft labels for the sample with a strong model. Only a starting point for review.

uv run python -m eval.metadata.prelabel [--model anthropic/claude-fable-5.1] [--max-cost 5]
"""

import argparse
import json
import re
from typing import Dict, List, Optional

from .common import (
    CATALOG_PATH,
    DRAFTS_PATH,
    FIELDS,
    SAMPLE_PATH,
    Usage,
    llm_input,
    load_env,
    openrouter,
    read_json,
    write_json,
)

DEFAULT_MODEL = "anthropic/claude-fable-5.1"
BATCH_SIZE = 10

RULES = """Label rules:
- Only values that are in the catalog go in the catalog fields, spelled exactly as in the catalog.
- If the post names several cameras or films, label the first one mentioned.
- A push or pull (HP5 @ 1600) keeps the box speed.
- A camera or film the post names that isn't in the catalog goes in unmatched, as written, and its
  catalog fields stay empty, or make only when the make is in the catalog.
- A different spelling of a catalog entry is the catalog entry (AE1 is ae-1, HP5+ is hp5 plus,
  N90 is f90 when only f90 is listed, Hasselblad 500CM is 500c/m).
- Lenses (Zeiss, Voigtlander 35mm, Nikkor), developers (D76, Rodinal), scanners and papers are
  not cameras or films.
- The title, description and the photographer's own comments all count.
- focal_length is an integer in mm, the first number of a range. aperture is "f/X" like "f/2.8".
- Leave a field null rather than guess."""

SYSTEM = """You label analog photography posts with the camera, film and lens they were shot on.

{rules}

Return JSON: {{"labels": [{{"post_id": 123, "camera_make": null, "camera_model": null,
"film_make": null, "film_type": null, "film_speed": null, "focal_length": null,
"aperture": null, "unmatched": [{{"kind": "camera", "raw": "Contax G2"}}],
"note": "short, only when unsure"}}]}}
One object per post, in order.

Catalog (make: entries). Cameras:
{cameras}

Films (make: type (speed)):
{films}"""


def catalog_text(catalog: Dict) -> tuple[str, str]:
    cameras: Dict[str, List[str]] = {}
    for c in catalog["cameras"]:
        cameras.setdefault(c["make"], []).append(c["model"])
    films: Dict[str, List[str]] = {}
    for f in catalog["films"]:
        films.setdefault(f["make"], []).append(f"{f['type']} ({f['speed']})")
    return (
        "\n".join(f"{m}: {' | '.join(v)}" for m, v in sorted(cameras.items())),
        "\n".join(f"{m}: {' | '.join(v)}" for m, v in sorted(films.items())),
    )


def parse_labels(content: Optional[str]) -> List[Dict]:
    if not content:
        return []
    content = content.strip()
    content = re.sub(r"^```(?:json)?\s*", "", content)
    content = re.sub(r"\s*```$", "", content)
    start = content.find("{")
    data = json.loads(content[start:] if start >= 0 else content)
    labels = data.get("labels") if isinstance(data, dict) else data
    return [x for x in labels or [] if isinstance(x, dict)]


def _lower(v) -> Optional[str]:
    return v.strip().lower() if isinstance(v, str) and v.strip() else None


def _int(v) -> Optional[int]:
    try:
        return int(v) if v is not None and v != "" else None
    except (TypeError, ValueError):
        return None


def clean_draft(label: Dict, catalog: Dict) -> Dict:
    """Keep only catalog values. Anything else moves to unmatched."""
    cameras = {(c["make"], c["model"]) for c in catalog["cameras"]}
    camera_makes = {c["make"] for c in catalog["cameras"]}
    films = {(f["make"], f["type"]): f["speed"] for f in catalog["films"]}
    film_makes = {f["make"] for f in catalog["films"]}
    unmatched = [
        {"kind": u.get("kind"), "raw": u.get("raw")}
        for u in label.get("unmatched") or []
        if isinstance(u, dict) and u.get("kind") in ("camera", "film") and u.get("raw")
    ]
    raws = {u["raw"].lower() for u in unmatched}

    def lost(kind: str, *parts):
        raw = " ".join(p for p in parts if p)
        if raw and raw.lower() not in raws:
            unmatched.append({"kind": kind, "raw": raw})
            raws.add(raw.lower())

    out = {f: None for f in FIELDS}
    make, model = _lower(label.get("camera_make")), _lower(label.get("camera_model"))
    if make in camera_makes:
        out["camera_make"] = make
        if model and (make, model) in cameras:
            out["camera_model"] = model
        elif model:
            lost("camera", make, model)
    elif make or model:
        lost("camera", make, model)

    fmake, ftype = _lower(label.get("film_make")), _lower(label.get("film_type"))
    if fmake in film_makes:
        out["film_make"] = fmake
        if ftype and (fmake, ftype) in films:
            out["film_type"] = ftype
        elif ftype:
            lost("film", fmake, ftype)
    elif fmake or ftype:
        lost("film", fmake, ftype)

    if out["film_type"]:
        out["film_speed"] = films[(out["film_make"], out["film_type"])]
    else:
        out["film_speed"] = _int(label.get("film_speed"))
    out["focal_length"] = _int(label.get("focal_length"))
    aperture = label.get("aperture")
    out["aperture"] = str(aperture).strip() if aperture else None
    out["unmatched"] = unmatched
    out["note"] = label.get("note") or ""
    return out


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--model", default=DEFAULT_MODEL)
    parser.add_argument("--max-cost", type=float, default=5.0)
    parser.add_argument(
        "--limit", type=int, default=0, help="only draft this many posts"
    )
    args = parser.parse_args()
    load_env()

    sample = read_json(SAMPLE_PATH)
    catalog = read_json(CATALOG_PATH)
    drafts = read_json(DRAFTS_PATH) if DRAFTS_PATH.exists() else {"labels": {}}
    todo = [p for p in sample["posts"] if str(p["id"]) not in drafts["labels"]]
    if args.limit:
        todo = todo[: args.limit]
    print(f"{len(todo)} posts to draft with {args.model}")

    cameras, films = catalog_text(catalog)
    system = SYSTEM.format(rules=RULES, cameras=cameras, films=films)
    client = openrouter()
    usage = Usage()

    for start in range(0, len(todo), BATCH_SIZE):
        if usage.cost >= args.max_cost:
            print(f"stopping at ${usage.cost:.2f}, over --max-cost")
            break
        batch = todo[start : start + BATCH_SIZE]
        prompt = "\n\n".join(f"post_id: {p['id']}\n{llm_input(p)}" for p in batch)
        try:
            resp = client.chat.completions.create(
                model=args.model,
                messages=[
                    {
                        "role": "system",
                        "content": [
                            {
                                "type": "text",
                                "text": system,
                                "cache_control": {"type": "ephemeral"},
                            }
                        ],
                    },
                    {"role": "user", "content": prompt},
                ],
                temperature=0,
                max_tokens=8000,
                extra_body={"reasoning": {"effort": "low"}},
            )
            usage.add(resp.usage)
            labels = parse_labels(resp.choices[0].message.content)
        except Exception as e:
            print(f"  batch at {start} failed: {e}")
            continue
        ids = {p["id"] for p in batch}
        for label in labels:
            try:
                post_id = int(label.get("post_id"))
            except (TypeError, ValueError):
                continue
            if post_id in ids:
                drafts["labels"][str(post_id)] = clean_draft(label, catalog)
        drafts["model"] = args.model
        write_json(DRAFTS_PATH, drafts)
        done = len(drafts["labels"])
        print(f"  {done}/{len(sample['posts'])} drafted, ${usage.cost:.3f} so far")

    print(f"usage: {usage.to_dict()}")


if __name__ == "__main__":
    main()
