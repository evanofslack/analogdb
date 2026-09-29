"""Run an extractor over the labeled set and write a report.

uv run python -m eval.metadata.score --extractor stored
uv run python -m eval.metadata.score --extractor current --tag baseline
uv run python -m eval.metadata.score --extractor v2 --tag v2 --compare baseline
uv run python -m eval.metadata.score --tag baseline --rescore   (no LLM calls, cached predictions)
"""

import argparse
import html
import os
import re
import time
from datetime import datetime, timezone
from typing import Callable, Dict, List, Optional, Tuple

from scrape.metadata import MetadataExtractor
from scrape.normalize import normalize_key

from .common import (
    CATALOG_PATH,
    DRAFTS_PATH,
    FIELDS,
    GOLD_PATH,
    REPORTS,
    RUNS,
    SAMPLE_PATH,
    TrackedOpenAI,
    Usage,
    catalog_models,
    load_env,
    openrouter,
    read_json,
    write_json,
)

Prediction = Dict  # {"fields": {...}, "unmatched": [{"kind", "raw"}], "flags": [...]}


def run_stored(posts: List[Dict], catalog: Dict, args, usage: Usage) -> Dict:
    """What the site has today. Pre-2025 posts are mostly empty."""
    return {
        p["id"]: {"fields": dict(p["stored"]), "unmatched": [], "flags": []}
        for p in posts
    }


def current_input(post: Dict) -> str:
    """The same text the ingest sends today: title and description, no comments."""
    text = f"title: {post['title']}"
    if post.get("description") is not None:
        text += f" description: {post['description']}"
    return text


def run_current(posts: List[Dict], catalog: Dict, args, usage: Usage) -> Dict:
    cameras, films = catalog_models(catalog)
    client = TrackedOpenAI(openrouter(), usage)
    extractor = MetadataExtractor(client, args.model, args.batch_size or 25)
    result = extractor.extract([current_input(p) for p in posts], films, cameras)
    preds = {}
    for p, m in zip(posts, result.metadata, strict=True):
        preds[p["id"]] = {
            "fields": {f: getattr(m, f) for f in FIELDS},
            "unmatched": [],
            "flags": [],
        }
    preds["_failed"] = result.failed
    return preds


EXTRACTORS: Dict[str, Callable] = {
    "stored": run_stored,
    "current": run_current,
}

# Extractors that report catalog misses, so unmatched recall means something
REPORTS_UNMATCHED: set = set()


def norm(field: str, v) -> Optional[object]:
    if v is None or v == "":
        return None
    if field in ("film_speed", "focal_length"):
        try:
            return int(v)
        except (TypeError, ValueError):
            return None
    if field == "aperture":
        m = re.search(r"(\d+(?:\.\d+)?)", str(v))
        return float(m.group(1)) if m else None
    return str(v).strip().lower()


def pair(make: str, second: str) -> Callable[[Dict], Optional[Tuple]]:
    def key(fields: Dict):
        b = norm(second, fields.get(second))
        return (norm(make, fields.get(make)), b) if b is not None else None

    return key


def single(field: str) -> Callable[[Dict], Optional[object]]:
    return lambda fields: norm(field, fields.get(field))


SCORED: Dict[str, Tuple[Callable, Optional[str]]] = {
    "camera": (pair("camera_make", "camera_model"), "camera"),
    "camera_make": (single("camera_make"), "camera"),
    "film": (pair("film_make", "film_type"), "film"),
    "film_make": (single("film_make"), "film"),
    "film_speed": (single("film_speed"), "film"),
    "focal_length": (single("focal_length"), None),
    "aperture": (single("aperture"), None),
}


def names_entity(gold: Dict, entity: Optional[str]) -> bool:
    """The post names a camera (or film) at all, in or out of the catalog."""
    if entity is None:
        return False
    fields = (
        ("camera_make", "camera_model")
        if entity == "camera"
        else ("film_make", "film_type", "film_speed")
    )
    if any(gold.get(f) for f in fields):
        return True
    return any(u.get("kind") == entity for u in gold.get("unmatched") or [])


def field_stats(
    pairs: List[Tuple[Dict, Dict]], key: Callable, entity: Optional[str] = None
) -> Dict:
    """pairs: (gold label, predicted fields) per post. A value set where gold is empty
    is wrong when the post names that camera or film some other way, else spurious."""
    s = {"gold": 0, "set": 0, "correct": 0, "wrong": 0, "spurious": 0}
    for gold, pred in pairs:
        g, p = key(gold), key(pred)
        if g is not None:
            s["gold"] += 1
        if p is None:
            continue
        s["set"] += 1
        if p == g:
            s["correct"] += 1
        elif g is None and not names_entity(gold, entity):
            s["spurious"] += 1
        else:
            s["wrong"] += 1
    s["precision"] = s["correct"] / s["set"] if s["set"] else None
    s["recall"] = s["correct"] / s["gold"] if s["gold"] else None
    s["wrong_rate"] = s["wrong"] / s["set"] if s["set"] else None
    return s


def mention_hit(gold: Dict, preds: List[Dict]) -> bool:
    g = normalize_key(gold["raw"])
    for p in preds:
        if p.get("kind") != gold["kind"]:
            continue
        k = normalize_key(p.get("raw") or p.get("key") or "")
        if k and (k == g or k in g or g in k):
            return True
    return False


def unmatched_recall(labels: List[Dict], preds: Dict) -> Optional[Dict]:
    total = hits = 0
    for label in labels:
        pred = preds.get(label["post_id"]) or {}
        for u in label.get("unmatched") or []:
            total += 1
            hits += mention_hit(u, pred.get("unmatched") or [])
    return {"gold": total, "hits": hits, "recall": hits / total if total else None}


def compute_metrics(labels: List[Dict], preds: Dict, with_unmatched: bool) -> Dict:
    def scored(subset):
        pairs = [
            (l, (preds.get(l["post_id"]) or {}).get("fields") or {}) for l in subset
        ]
        return {
            name: field_stats(pairs, key, entity)
            for name, (key, entity) in SCORED.items()
        }

    strata: Dict[str, List[Dict]] = {}
    for label in labels:
        strata.setdefault(label["stratum"], []).append(label)
    metrics = {
        "posts": len(labels),
        "fields": scored(labels),
        "strata": {
            name: {"posts": len(subset), "fields": scored(subset)}
            for name, subset in sorted(strata.items())
        },
        "unmatched": unmatched_recall(labels, preds) if with_unmatched else None,
    }
    flags: Dict[str, int] = {}
    for label in labels:
        for f in (preds.get(label["post_id"]) or {}).get("flags") or []:
            flags[f] = flags.get(f, 0) + 1
    metrics["flags"] = flags
    return metrics


def pct(v: Optional[float]) -> str:
    return "–" if v is None else f"{v * 100:.1f}%"


def delta(v: Optional[float], old: Optional[float]) -> str:
    if v is None or old is None:
        return ""
    d = (v - old) * 100
    return f"{d:+.1f}"


def field_table(metrics: Dict, old: Optional[Dict]) -> List[List[str]]:
    head = [
        "field",
        "gold",
        "set",
        "correct",
        "wrong",
        "spurious",
        "precision",
        "recall",
        "wrong %",
    ]
    if old:
        head += ["Δ precision", "Δ recall", "Δ wrong"]
    rows = [head]
    for name, s in metrics["fields"].items():
        row = [
            name,
            str(s["gold"]),
            str(s["set"]),
            str(s["correct"]),
            str(s["wrong"]),
            str(s["spurious"]),
            pct(s["precision"]),
            pct(s["recall"]),
            pct(s["wrong_rate"]),
        ]
        if old:
            o = old["fields"].get(name, {})
            row += [
                delta(s["precision"], o.get("precision")),
                delta(s["recall"], o.get("recall")),
                delta(s["wrong_rate"], o.get("wrong_rate")),
            ]
        rows.append(row)
    return rows


def strata_table(metrics: Dict) -> List[List[str]]:
    rows = [
        [
            "stratum",
            "posts",
            "camera P",
            "camera R",
            "camera wrong",
            "film P",
            "film R",
            "film wrong",
        ]
    ]
    for name, s in metrics["strata"].items():
        c, f = s["fields"]["camera"], s["fields"]["film"]
        rows.append(
            [
                name,
                str(s["posts"]),
                pct(c["precision"]),
                pct(c["recall"]),
                str(c["wrong"]),
                pct(f["precision"]),
                pct(f["recall"]),
                str(f["wrong"]),
            ]
        )
    return rows


def markdown_table(rows: List[List[str]]) -> str:
    out = ["| " + " | ".join(rows[0]) + " |", "|" + "---|" * len(rows[0])]
    out += ["| " + " | ".join(r) + " |" for r in rows[1:]]
    return "\n".join(out)


def summary_lines(run: Dict, metrics: Dict) -> List[str]:
    u = run["usage"]
    lines = [
        f"- extractor: `{run['extractor']}`, model: `{run.get('model') or '–'}`",
        f"- gold: `{run['gold']}`, posts scored: {metrics['posts']}",
        f"- LLM failures: {run.get('failed', 0)}, calls: {u['calls']}, "
        f"tokens in/out: {u['input_tokens']:,}/{u['output_tokens']:,}, cost: ${u['cost']:.3f}, "
        f"time: {run['seconds']:.0f}s",
    ]
    if metrics["unmatched"]:
        m = metrics["unmatched"]
        lines.append(
            f"- unmatched recall: {pct(m['recall'])} ({m['hits']}/{m['gold']})"
        )
    if metrics["flags"]:
        lines.append(
            "- flags: "
            + ", ".join(f"{k} {v}" for k, v in sorted(metrics["flags"].items()))
        )
    return lines


def markdown_report(
    tag: str, run: Dict, metrics: Dict, old: Optional[Dict], compare: Optional[str]
) -> str:
    parts = [f"# Metadata eval: {tag}", ""]
    parts += summary_lines(run, metrics)
    if compare:
        parts.append(f"- compared with: `{compare}`")
    parts += [
        "",
        "Precision: of the values set, share equal to gold. Recall: of gold values, share set "
        "correctly. Wrong: set, and different from a non-empty gold value. Spurious: set where "
        "gold is empty. `camera` and `film` are make and model or type together.",
        "",
        "## Fields",
        "",
        markdown_table(field_table(metrics, old)),
        "",
        "## Strata",
        "",
        markdown_table(strata_table(metrics)),
        "",
    ]
    return "\n".join(parts)


def disagreements(labels: List[Dict], preds: Dict) -> List[Tuple[int, Dict, List[str]]]:
    """Posts where any field differs, worst first (wrong, then spurious, then missed)."""
    out = []
    for label in labels:
        pred = (preds.get(label["post_id"]) or {}).get("fields") or {}
        rank, bad = 3, []
        for f in FIELDS:
            g, p = norm(f, label.get(f)), norm(f, pred.get(f))
            if g == p:
                continue
            bad.append(f)
            rank = min(
                rank,
                0 if (g is not None and p is not None) else 1 if p is not None else 2,
            )
        missed = [
            u
            for u in label.get("unmatched") or []
            if not mention_hit(
                u, (preds.get(label["post_id"]) or {}).get("unmatched") or []
            )
        ]
        if bad or missed:
            out.append((rank, label, bad))
    out.sort(key=lambda x: (x[0], x[1]["post_id"]))
    return out


def html_table(rows: List[List[str]]) -> str:
    head = "".join(f"<th>{html.escape(c)}</th>" for c in rows[0])
    body = "".join(
        "<tr>" + "".join(f"<td>{html.escape(c)}</td>" for c in r) + "</tr>"
        for r in rows[1:]
    )
    return f"<table><thead><tr>{head}</tr></thead><tbody>{body}</tbody></table>"


def html_report(
    tag: str,
    run: Dict,
    metrics: Dict,
    old: Optional[Dict],
    compare: Optional[str],
    labels: List[Dict],
    posts: Dict[int, Dict],
    other: Optional[Dict],
) -> str:
    preds = run["predictions"]
    cards = []
    kinds = ["wrong", "spurious", "missed"]
    for rank, label, bad in disagreements(labels, preds):
        pid = label["post_id"]
        post = posts[pid]
        pred = preds.get(pid) or {}
        opred = (other or {}).get(pid) or {}
        head = "<tr><th>field</th><th>gold</th><th>" + html.escape(tag) + "</th>"
        head += f"<th>{html.escape(compare)}</th>" if other is not None else ""
        rows = [head + "</tr>"]
        for f in FIELDS:
            g, p = label.get(f), (pred.get("fields") or {}).get(f)
            cls = "bad" if f in bad else ""
            row = f"<tr class='{cls}'><td>{f}</td><td>{html.escape(str(g or ''))}</td><td>{html.escape(str(p or ''))}</td>"
            if other is not None:
                row += f"<td>{html.escape(str((opred.get('fields') or {}).get(f) or ''))}</td>"
            rows.append(row + "</tr>")
        extra = []
        if label.get("unmatched"):
            extra.append(
                "gold unmatched: " + "; ".join(u["raw"] for u in label["unmatched"])
            )
        if pred.get("unmatched"):
            extra.append(
                "predicted unmatched: "
                + "; ".join(
                    str(u.get("raw") or u.get("key")) for u in pred["unmatched"]
                )
            )
        if pred.get("flags"):
            extra.append("flags: " + ", ".join(pred["flags"]))
        if label.get("note"):
            extra.append("note: " + label["note"])
        text = post["title"] + (
            f" · {post['description']}" if post.get("description") else ""
        )
        cards.append(
            f"""<div class="card"><div class="head"><b>#{pid}</b> {html.escape(label["stratum"])}
            · <span class="k{rank}">{kinds[min(rank, 2)]}</span>
            · <a href="https://analogdb.com/post/{pid}" target="_blank" rel="noopener">post</a></div>
            <div class="top"><img loading="lazy" src="{html.escape(post.get("image") or "")}" alt="">
            <div><div class="title">{html.escape(text[:600])}</div>
            <table class="f">{"".join(rows)}</table>
            {"".join(f'<div class="x">{html.escape(e)}</div>' for e in extra)}</div></div></div>"""
        )
    summary = "".join(
        f"<li>{html.escape(l.lstrip('- '))}</li>" for l in summary_lines(run, metrics)
    )
    return f"""<!doctype html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1"><title>Eval {html.escape(tag)}</title>
<style>
:root {{ --bg:#fafafa; --fg:#1a1a1a; --card:#fff; --muted:#666; --line:#ddd; --bad:#fee2e2; --warn:#b45309; }}
@media (prefers-color-scheme: dark) {{ :root {{ --bg:#111; --fg:#eee; --card:#1c1c1c; --muted:#999; --line:#333; --bad:#3b1414; --warn:#fbbf24; }} }}
body {{ background:var(--bg); color:var(--fg); font:14px/1.4 system-ui,sans-serif; margin:0; padding:16px; }}
h1 {{ font-size:20px; }} h2 {{ font-size:16px; margin-top:24px; }}
table {{ border-collapse:collapse; }} th, td {{ padding:3px 8px; border-bottom:1px solid var(--line); text-align:left; }}
th {{ color:var(--muted); font-weight:500; }}
.wrap {{ overflow-x:auto; }}
.cards {{ display:grid; gap:12px; max-width:900px; }}
.card {{ background:var(--card); border:1px solid var(--line); border-radius:8px; padding:10px; }}
.head {{ color:var(--muted); font-size:12px; margin-bottom:6px; }} .head b {{ color:var(--fg); }}
.k0 {{ color:#dc2626; font-weight:600; }} .k1 {{ color:var(--warn); }}
.top {{ display:grid; grid-template-columns:180px 1fr; gap:10px; }} .top img {{ width:100%; border-radius:4px; }}
.title {{ font-weight:600; margin-bottom:6px; overflow-wrap:anywhere; }}
table.f {{ width:100%; font-size:13px; }} tr.bad td {{ background:var(--bad); }}
.x {{ color:var(--muted); font-size:12px; margin-top:4px; }}
@media (max-width:600px) {{ .top {{ grid-template-columns:1fr; }} }}
</style></head><body>
<h1>Metadata eval: {html.escape(tag)}</h1><ul>{summary}</ul>
<h2>Fields</h2><div class="wrap">{html_table(field_table(metrics, old))}</div>
<h2>Strata</h2><div class="wrap">{html_table(strata_table(metrics))}</div>
<h2>Disagreements ({len(cards)})</h2><div class="cards">{"".join(cards)}</div>
</body></html>"""


def load_gold(source: str, which: str) -> List[Dict]:
    """Reviewed gold labels, or the unreviewed drafts for a smoke test."""
    sample = {p["id"]: p for p in read_json(SAMPLE_PATH)["posts"]}
    if source == "drafts":
        drafts = read_json(DRAFTS_PATH)["labels"]
        labels = [
            {
                **d,
                "post_id": int(pid),
                "set": sample[int(pid)]["set"],
                "stratum": sample[int(pid)]["stratum"],
            }
            for pid, d in drafts.items()
        ]
    else:
        labels = [l for l in read_json(GOLD_PATH)["labels"] if l.get("reviewed")]
    return [l for l in labels if l["set"] == which and l["post_id"] in sample]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--extractor", choices=sorted(EXTRACTORS), default="current")
    parser.add_argument("--model", default=None, help="default: OPENROUTER_MODEL")
    parser.add_argument("--batch-size", type=int, default=0)
    parser.add_argument(
        "--tag", default=None, help="report name, default: the extractor"
    )
    parser.add_argument("--compare", default=None, help="tag of an earlier report")
    parser.add_argument("--gold", choices=["gold", "drafts"], default="gold")
    parser.add_argument(
        "--set", dest="which", choices=["test", "fewshot"], default="test"
    )
    parser.add_argument(
        "--rescore", action="store_true", help="reuse cached predictions"
    )
    args = parser.parse_args()
    load_env()
    args.model = args.model or os.environ.get("OPENROUTER_MODEL")
    tag = args.tag or args.extractor

    labels = load_gold(args.gold, args.which)
    sample = {p["id"]: p for p in read_json(SAMPLE_PATH)["posts"]}
    posts = [sample[l["post_id"]] for l in labels]
    catalog = read_json(CATALOG_PATH)
    run_path = RUNS / f"{tag}.json"

    if args.rescore:
        run = read_json(run_path)
        run["predictions"] = {int(k): v for k, v in run["predictions"].items()}
        missing = [p["id"] for p in posts if p["id"] not in run["predictions"]]
        if missing:
            raise SystemExit(
                f"cached run {tag} lacks {len(missing)} posts, run without --rescore"
            )
    else:
        usage = Usage()
        start = time.time()
        preds = EXTRACTORS[args.extractor](posts, catalog, args, usage)
        failed = preds.pop("_failed", 0)
        run = {
            "tag": tag,
            "extractor": args.extractor,
            "model": args.model if args.extractor != "stored" else None,
            "created": int(datetime.now(timezone.utc).timestamp()),
            "failed": failed,
            "usage": usage.to_dict(),
            "seconds": time.time() - start,
            "predictions": preds,
        }
        write_json(run_path, run)
    run["gold"] = args.gold

    with_unmatched = run["extractor"] in REPORTS_UNMATCHED
    metrics = compute_metrics(labels, run["predictions"], with_unmatched)
    old = other = None
    if args.compare:
        old = read_json(REPORTS / f"{args.compare}.json")["metrics"]
        other_path = RUNS / f"{args.compare}.json"
        if other_path.exists():
            other = {int(k): v for k, v in read_json(other_path)["predictions"].items()}

    write_json(REPORTS / f"{tag}.json", {"tag": tag, "metrics": metrics})
    (REPORTS / f"{tag}.md").write_text(
        markdown_report(tag, run, metrics, old, args.compare)
    )
    (REPORTS / f"{tag}.html").write_text(
        html_report(tag, run, metrics, old, args.compare, labels, sample, other)
    )
    print(markdown_report(tag, run, metrics, old, args.compare))
    print(f"wrote {REPORTS / tag}.md and .html")


if __name__ == "__main__":
    main()
