"""Compare vision models for image tagging on a sample of posts.

Runs the scraper's own ImageTagger once per model and post, records cost,
latency and refusals, and writes a static review page with each model's
caption and tags side by side.

Usage:
    uv run python scripts/eval_tags.py --sample-only
    uv run python scripts/eval_tags.py --models google/gemini-2.5-flash-lite google/gemini-2.5-flash
    uv run python scripts/eval_tags.py --render-only

Requires OPENROUTER_API_KEY (and optionally OPENROUTER_BASE_URL) for tagging.
Everything is written to eval/tags/data/, which is not committed.
"""

import argparse
import html
import json
import os
import re
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
from types import SimpleNamespace
from typing import Callable, Dict, List, Optional

import requests
from scrape.tagging import (
    ImageTagger,
    catalog_words,
    load_stoplist,
    medium_url,
)

API = "https://api.analogdb.com/v1"
ROOT = Path(__file__).resolve().parents[1]
DATA_DIR = ROOT / "eval" / "tags" / "data"
DEFAULT_MODELS = ["google/gemini-2.5-flash-lite", "google/gemini-2.5-flash"]
SEEDS = [11, 23, 37, 41, 59, 67, 71, 89, 97, 101]
PLACE = re.compile(r"\b(?:in|at|from|near|around|of)\s+(?:the\s+)?[A-Z][a-zA-Z]{2,}")

Fetch = Callable[[dict], List[dict]]


def fetch_posts(params: dict) -> List[dict]:
    resp = requests.get(f"{API}/posts", params=params, timeout=30)
    resp.raise_for_status()
    return resp.json().get("posts") or []


def sample_posts(
    fetch: Fetch, nsfw: int = 10, bw: int = 10, places: int = 10, total: int = 60
) -> List[dict]:
    """NSFW, black and white and place-name posts first, then random posts to
    fill the total. Each post is kept once."""
    seen: set = set()
    out: List[dict] = []

    def take(group: str, count: int, params: dict, keep=lambda p: True) -> None:
        found = 0
        for seed in SEEDS:
            if found >= count:
                return
            page = fetch(dict(params, sort="random", seed=seed, page_size=200))
            for p in page:
                if found >= count:
                    return
                if p["id"] in seen or not keep(p):
                    continue
                try:
                    url = medium_url(
                        [SimpleNamespace(**i) for i in p.get("images") or []]
                    )
                except (ValueError, TypeError):
                    continue
                seen.add(p["id"])
                found += 1
                out.append(
                    {
                        "id": p["id"],
                        "group": group,
                        "title": p.get("title") or "",
                        "description": p.get("description"),
                        "url": url,
                        "nsfw": bool(p.get("nsfw")),
                        "grayscale": bool(p.get("grayscale")),
                        "keywords": [k["word"] for k in p.get("keywords") or []],
                    }
                )

    take("nsfw", nsfw, {"nsfw": "true"})
    take("bw", bw, {"grayscale": "true", "nsfw": "false"})
    take(
        "place",
        places,
        {"nsfw": "false"},
        lambda p: bool(PLACE.search(p.get("title") or "")),
    )
    take("random", total - len(out), {})
    return out


class UsageRecorder:
    """Wraps an OpenAI client and records usage and refusals per call, per
    thread, so concurrent tagging keeps each post's calls apart."""

    def __init__(self, openai):
        self.openai = openai
        self.local = threading.local()
        self.chat = SimpleNamespace(completions=SimpleNamespace(create=self._create))

    def start(self) -> None:
        self.local.calls = []

    def calls(self) -> List[Dict]:
        return list(getattr(self.local, "calls", []))

    def _create(self, **kwargs):
        extra = dict(kwargs.pop("extra_body", None) or {})
        extra["usage"] = {"include": True}
        call: Dict = {"cost": None, "prompt_tokens": None, "completion_tokens": None}
        if not hasattr(self.local, "calls"):
            self.local.calls = []
        self.local.calls.append(call)
        try:
            resp = self.openai.chat.completions.create(extra_body=extra, **kwargs)
        except Exception as e:
            call["error"] = f"{type(e).__name__}: {e}"[:300]
            raise
        usage = getattr(resp, "usage", None)
        if usage is not None:
            call["cost"] = getattr(usage, "cost", None)
            call["prompt_tokens"] = getattr(usage, "prompt_tokens", None)
            call["completion_tokens"] = getattr(usage, "completion_tokens", None)
        choice = resp.choices[0]
        call["finish_reason"] = getattr(choice, "finish_reason", None)
        call["refusal"] = bool(getattr(choice.message, "refusal", None))
        call["image"] = any(
            isinstance(part, dict) and part.get("type") == "image_url"
            for m in kwargs.get("messages", [])
            if isinstance(m.get("content"), list)
            for part in m["content"]
        )
        return resp


def tag_post(tagger: ImageTagger, recorder: UsageRecorder, post: dict) -> Dict:
    recorder.start()
    started = time.monotonic()
    result: Dict = {}
    try:
        tags = tagger.tag(post["url"], post["title"], post.get("description"))
        result.update(
            caption=tags.caption, tags=tags.tags, version=tags.version, raw=tags.raw
        )
    except Exception as e:
        result.update(caption=None, tags=[], version=None, error=str(e))
    calls = recorder.calls()
    costs = [c["cost"] for c in calls if c.get("cost") is not None]
    result.update(
        seconds=round(time.monotonic() - started, 2),
        calls=calls,
        cost=sum(costs) if costs else None,
    )
    return result


def run_model(
    openai,
    model: str,
    posts: List[dict],
    blocked: set,
    done: Dict[str, Dict],
    concurrency: int,
    save: Callable[[str, Dict], None],
) -> Dict[str, Dict]:
    recorder = UsageRecorder(openai)
    tagger = ImageTagger(recorder, model, blocked)
    todo = [p for p in posts if str(p["id"]) not in done]
    results = dict(done)

    def work(post: dict):
        return str(post["id"]), tag_post(tagger, recorder, post)

    with ThreadPoolExecutor(max_workers=max(concurrency, 1)) as pool:
        for i, (id, result) in enumerate(pool.map(work, todo), start=1):
            results[id] = result
            save(id, result)
            print(
                f"  {model}: {i}/{len(todo)} post {id} {result.get('version') or 'failed'}"
            )
    return results


def summarize(
    posts: List[dict], results: Dict[str, Dict[str, Dict]]
) -> Dict[str, Dict]:
    groups = {str(p["id"]): p["group"] for p in posts}
    summary = {}
    for model, by_post in results.items():
        rows = [r for id, r in by_post.items() if id in groups]
        nsfw = [r for id, r in by_post.items() if groups.get(id) == "nsfw"]
        text_only = sum(1 for r in rows if (r.get("version") or "").endswith("-text"))
        failed = sum(1 for r in rows if r.get("version") is None)
        image_refused = sum(
            1
            for r in rows
            if any(
                c.get("image") and (c.get("refusal") or c.get("error"))
                for c in r["calls"]
            )
        )
        costs = [r["cost"] for r in rows if r.get("cost") is not None]
        n = len(rows) or 1
        summary[model] = {
            "posts": len(rows),
            "image_ok": len(rows) - text_only - failed,
            "text_only": text_only,
            "failed": failed,
            "refusal_rate": round((text_only + failed) / n, 3),
            "nsfw_refusal_rate": round(
                sum(1 for r in nsfw if (r.get("version") or "-text").endswith("-text"))
                / (len(nsfw) or 1),
                3,
            ),
            "posts_with_image_retry_or_refusal": image_refused,
            "mean_tags": round(sum(len(r["tags"]) for r in rows) / n, 1),
            "cost_total": round(sum(costs), 6) if costs else None,
            "cost_per_image": round(sum(costs) / len(costs), 6) if costs else None,
            "mean_seconds": round(sum(r["seconds"] for r in rows) / n, 2),
        }
    return summary


def render(
    posts: List[dict], results: Dict[str, Dict[str, Dict]], summary: Dict, out: Path
) -> None:
    models = list(results)
    esc = html.escape

    def column(model: str, post: dict) -> str:
        r = results[model].get(str(post["id"]))
        if r is None:
            return f'<div class="col"><h4>{esc(model)}</h4><p class="muted">not run</p></div>'
        version = r.get("version") or "failed"
        chips = "".join(
            f'<button class="chip" data-tag="{esc(t)}">{esc(t)}</button>'
            for t in r["tags"]
        )
        cost = f"${r['cost']:.5f}" if r.get("cost") is not None else "cost n/a"
        error = f'<p class="error">{esc(r["error"])}</p>' if r.get("error") else ""
        return f"""<div class="col" data-model="{esc(model)}">
  <h4>{esc(model)} <span class="muted">{esc(version)} · {cost} · {r['seconds']}s</span></h4>
  <p>{esc(r.get('caption') or '(no caption)')}</p>
  <div class="chips">{chips}</div>{error}
  <input class="missing" placeholder="missing tags, comma separated">
</div>"""

    cards = []
    for p in posts:
        cols = "".join(column(m, p) for m in models)
        old = ", ".join(p.get("keywords") or [])
        cards.append(
            f"""<section class="card" data-id="{p['id']}">
  <a href="https://analogdb.com/post/{p['id']}" target="_blank"><img loading="lazy" src="{esc(p['url'])}" alt=""></a>
  <div class="body">
    <p><b>#{p['id']}</b> · {esc(p['group'])} · {esc(p['title'])}</p>
    <p class="muted">old keywords: {esc(old) or '(none)'}</p>
    <div class="cols">{cols}</div>
  </div>
</section>"""
        )

    rows = "".join(
        f"<tr><td>{esc(m)}</td>"
        + "".join(f"<td>{esc(str(v))}</td>" for v in s.values())
        + "</tr>"
        for m, s in summary.items()
    )
    head = "".join(f"<th>{esc(k)}</th>" for k in next(iter(summary.values()), {}))
    page = f"""<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Tag Review</title>
<style>
:root {{ --bg:#fafafa; --fg:#1a1a1a; --card:#fff; --muted:#666; --line:#ddd; --accent:#2563eb; --bad:#dc2626; }}
@media (prefers-color-scheme: dark) {{ :root {{ --bg:#111; --fg:#eee; --card:#1c1c1c; --muted:#999; --line:#333; --accent:#60a5fa; --bad:#f87171; }} }}
body {{ background:var(--bg); color:var(--fg); font:14px/1.4 system-ui,sans-serif; margin:0; padding:16px; }}
h1 {{ font-size:20px; margin:0 0 4px; }} h4 {{ margin:0 0 4px; font-size:13px; }}
p {{ margin:4px 0; }} .muted {{ color:var(--muted); font-weight:normal; }} .error {{ color:var(--bad); }}
.bar {{ position:sticky; top:0; background:var(--bg); padding:8px 0; border-bottom:1px solid var(--line); z-index:1; display:flex; gap:12px; align-items:center; }}
button#copy {{ background:var(--accent); color:#fff; border:0; border-radius:6px; padding:8px 14px; font:inherit; cursor:pointer; }}
table {{ border-collapse:collapse; margin:12px 0; font-size:12px; display:block; overflow-x:auto; }}
td, th {{ border:1px solid var(--line); padding:4px 6px; text-align:left; }}
.card {{ background:var(--card); border:1px solid var(--line); border-radius:8px; margin:12px 0; display:flex; flex-wrap:wrap; overflow:hidden; }}
.card img {{ width:320px; max-width:100%; height:320px; object-fit:cover; display:block; }}
.body {{ flex:1; min-width:260px; padding:8px 12px; }}
.cols {{ display:grid; grid-template-columns:repeat(auto-fit,minmax(240px,1fr)); gap:12px; margin-top:8px; }}
.chips {{ display:flex; flex-wrap:wrap; gap:4px; margin:6px 0; }}
.chip {{ border:1px solid var(--line); background:transparent; color:var(--fg); border-radius:12px; padding:2px 8px; font:inherit; font-size:12px; cursor:pointer; }}
.chip.wrong {{ border-color:var(--bad); color:var(--bad); text-decoration:line-through; }}
.missing {{ width:100%; box-sizing:border-box; font:inherit; font-size:12px; padding:4px; background:transparent; color:var(--fg); border:1px solid var(--line); border-radius:4px; }}
</style></head><body>
<h1>Tag review</h1>
<p class="muted">Click a tag to mark it wrong. Type missing tags under each model. Press Copy labels and paste the result to Claude. Choices are saved in this browser.</p>
<table><tr><th>model</th>{head}</tr>{rows}</table>
<div class="bar"><button id="copy">Copy labels</button><span id="status"></span></div>
{''.join(cards)}
<script>
const KEY = "tag-review-labels";
let labels = {{}};
try {{ labels = JSON.parse(localStorage.getItem(KEY) || "{{}}"); }} catch (e) {{}}
function save() {{ try {{ localStorage.setItem(KEY, JSON.stringify(labels)); }} catch (e) {{}} }}
function entry(id, model) {{ labels[id] = labels[id] || {{}}; labels[id][model] = labels[id][model] || {{ wrong: [], missing: "" }}; return labels[id][model]; }}
document.querySelectorAll(".card").forEach(card => {{
  const id = card.dataset.id;
  card.querySelectorAll(".col[data-model]").forEach(col => {{
    const model = col.dataset.model;
    const saved = (labels[id] || {{}})[model];
    col.querySelectorAll(".chip").forEach(chip => {{
      if (saved && saved.wrong.includes(chip.dataset.tag)) chip.classList.add("wrong");
      chip.addEventListener("click", () => {{
        const e = entry(id, model);
        chip.classList.toggle("wrong");
        e.wrong = [...col.querySelectorAll(".chip.wrong")].map(c => c.dataset.tag);
        save();
      }});
    }});
    const input = col.querySelector(".missing");
    if (saved) input.value = saved.missing;
    input.addEventListener("input", () => {{ entry(id, model).missing = input.value; save(); }});
  }});
}});
document.getElementById("copy").addEventListener("click", async () => {{
  const text = JSON.stringify(labels);
  try {{ await navigator.clipboard.writeText(text); document.getElementById("status").textContent = "Copied"; }}
  catch (e) {{ prompt("Copy labels", text); }}
}});
</script></body></html>"""
    out.write_text(page, encoding="utf-8")


def load_json(path: Path, default):
    if not path.exists():
        return default
    return json.loads(path.read_text(encoding="utf-8"))


def main(
    argv: Optional[List[str]] = None, fetch: Fetch = fetch_posts, openai=None
) -> Dict:
    parser = argparse.ArgumentParser(
        description="Compare vision models for image tagging"
    )
    parser.add_argument("--models", nargs="+", default=DEFAULT_MODELS)
    parser.add_argument("--work-dir", default=str(DATA_DIR))
    parser.add_argument("--total", type=int, default=60)
    parser.add_argument("--per-group", type=int, default=10)
    parser.add_argument("--concurrency", type=int, default=4)
    parser.add_argument(
        "--sample-only", action="store_true", help="pick the posts, no LLM calls"
    )
    parser.add_argument(
        "--render-only", action="store_true", help="rebuild the page from saved results"
    )
    args = parser.parse_args(argv)

    work = Path(args.work_dir)
    work.mkdir(parents=True, exist_ok=True)
    sample_path = work / "sample.json"
    results_path = work / "results.json"

    posts = load_json(sample_path, None)
    if posts is None:
        g = args.per_group
        posts = sample_posts(fetch, nsfw=g, bw=g, places=g, total=args.total)
        sample_path.write_text(json.dumps(posts, indent=2), encoding="utf-8")
    print(f"{len(posts)} posts in {sample_path}")

    results: Dict[str, Dict[str, Dict]] = load_json(results_path, {})
    if not args.sample_only and not args.render_only:
        if openai is None:
            from openai import OpenAI

            openai = OpenAI(
                base_url=os.environ.get(
                    "OPENROUTER_BASE_URL", "https://openrouter.ai/api/v1"
                ),
                api_key=os.environ["OPENROUTER_API_KEY"],
            )
        blocked = catalog_words(
            load_json(ROOT / "data" / "cameras.json", []),
            load_json(ROOT / "data" / "films.json", []),
        ) | load_stoplist(str(ROOT / "data" / "tag_stoplist.txt"))
        lock = threading.Lock()
        for model in args.models:

            def save(id: str, result: Dict, model: str = model) -> None:
                with lock:
                    results.setdefault(model, {})[id] = result
                    results_path.write_text(
                        json.dumps(results, indent=2), encoding="utf-8"
                    )

            run_model(
                openai,
                model,
                posts,
                blocked,
                results.get(model, {}),
                args.concurrency,
                save,
            )

    if args.sample_only:
        return {}
    summary = summarize(posts, results)
    (work / "summary.json").write_text(json.dumps(summary, indent=2), encoding="utf-8")
    render(posts, results, summary, work / "review.html")
    print(json.dumps(summary, indent=2))
    print(f"Review page: {work / 'review.html'}")
    return summary


if __name__ == "__main__":
    main()
