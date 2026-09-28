import argparse
import html
import io
import json
import os
from concurrent.futures import ThreadPoolExecutor

import numpy as np
import requests
from PIL import Image
from scrape.image import channel_spread

API = "https://api.analogdb.com/v1"
THRESHOLDS = range(0, 81)


def fetch_posts(params: dict) -> list:
    resp = requests.get(f"{API}/posts", params=params, timeout=30)
    resp.raise_for_status()
    return resp.json().get("posts") or []


SEEDS = [1000003, 1000033, 1000037, 1000039, 1000081, 1000099, 1000117, 1000121]


def sample(per_class: int, bw: int, hard: int) -> list:
    groups = [
        ("bw", bw, {"grayscale": "true"}),
        ("color", per_class, {"grayscale": "false"}),
        ("muted", hard, {"grayscale": "false", "color": "gray", "min_color": 0.6}),
    ]
    seen, posts = set(), []
    for group, count, params in groups:
        found = []
        for seed in SEEDS:
            if len(found) >= count:
                break
            found += fetch_posts(dict(params, sort="random", seed=seed, page_size=min(200, count)))
        for p in found[:count]:
            if p["id"] in seen:
                continue
            seen.add(p["id"])
            low = next(i for i in p["images"] if i["resolution"] == "low")
            posts.append({
                "id": p["id"],
                "group": group,
                "db_grayscale": bool(p.get("grayscale")),
                "title": p.get("title") or "",
                "url": low["url"],
                "permalink": p.get("permalink") or "",
            })
    return posts


def measure(post: dict, cache_dir: str) -> dict:
    path = os.path.join(cache_dir, f"{post['id']}.jpg")
    if not os.path.exists(path):
        resp = requests.get(post["url"], timeout=60)
        resp.raise_for_status()
        with open(path, "wb") as f:
            f.write(resp.content)
    with open(path, "rb") as f:
        spread = channel_spread(Image.open(io.BytesIO(f.read())))
    post["p99"] = float(np.percentile(spread, 99))
    post["p95"] = float(np.percentile(spread, 95))
    post["median"] = float(np.median(spread))
    return post


def best_threshold(posts: list, truth_key: str) -> tuple:
    scored = []
    for t in THRESHOLDS:
        errors = sum((p["p99"] <= t) != p[truth_key] for p in posts)
        scored.append((errors, t))
    errors, t = min(scored)
    return t, errors, scored


def render(posts: list, threshold: int, band: tuple, out: str) -> None:
    for p in posts:
        p["predicted"] = p["p99"] <= threshold
    review = [p for p in posts if p["predicted"] != p["db_grayscale"] or band[0] <= p["p99"] <= band[1]]
    rest = [p for p in posts if p not in review]
    review.sort(key=lambda p: p["p99"])
    rest.sort(key=lambda p: p["p99"])

    def card(p: dict) -> str:
        label = "bw" if p["predicted"] else "color"
        return f"""
<div class="card" data-id="{p['id']}" data-default="{label}">
  <a href="https://analogdb.com/post/{p['id']}" target="_blank"><img loading="lazy" src="{html.escape(p['url'])}"></a>
  <div class="meta">
    <b>#{p['id']}</b> · p99 {p['p99']:.0f} · p95 {p['p95']:.0f} · med {p['median']:.0f}<br>
    db: {'bw' if p['db_grayscale'] else 'color'} · predicted: {label} · {p['group']}
  </div>
  <div class="toggle">
    <label><input type="radio" name="l{p['id']}" value="bw"> B&amp;W</label>
    <label><input type="radio" name="l{p['id']}" value="color"> Colour</label>
  </div>
</div>"""

    page = f"""<!doctype html>
<html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Grayscale Review</title>
<style>
:root {{ --bg:#fafafa; --fg:#1a1a1a; --card:#fff; --muted:#666; --line:#ddd; --accent:#2563eb; }}
@media (prefers-color-scheme: dark) {{ :root {{ --bg:#111; --fg:#eee; --card:#1c1c1c; --muted:#999; --line:#333; --accent:#60a5fa; }} }}
body {{ background:var(--bg); color:var(--fg); font:14px/1.4 system-ui,sans-serif; margin:0; padding:16px; }}
h1 {{ font-size:20px; margin:0 0 4px; }} h2 {{ font-size:16px; margin:24px 0 8px; }}
p {{ color:var(--muted); margin:4px 0 12px; max-width:70ch; }}
.bar {{ position:sticky; top:0; background:var(--bg); padding:8px 0; border-bottom:1px solid var(--line); z-index:1; display:flex; gap:12px; align-items:center; flex-wrap:wrap; }}
button {{ background:var(--accent); color:#fff; border:0; border-radius:6px; padding:8px 14px; font:inherit; cursor:pointer; }}
.grid {{ display:grid; grid-template-columns:repeat(auto-fill,minmax(220px,1fr)); gap:12px; }}
.card {{ background:var(--card); border:1px solid var(--line); border-radius:8px; overflow:hidden; }}
.card.changed {{ outline:2px solid var(--accent); }}
.card img {{ width:100%; height:220px; object-fit:cover; display:block; }}
.meta {{ padding:6px 8px; color:var(--muted); font-size:12px; }}
.toggle {{ padding:0 8px 8px; display:flex; gap:12px; }}
details summary {{ cursor:pointer; margin:24px 0 8px; font-weight:600; }}
</style></head><body>
<h1>Grayscale review</h1>
<p>Only true black and white counts as grayscale. Sepia, toned and tinted prints are colour.
Each card is preset to the prediction at threshold <b>p99 ≤ {threshold}</b>. Fix any that are wrong, then press Copy labels and paste the result to Claude.
Choices are saved in this browser.</p>
<div class="bar"><button id="copy">Copy labels</button><span id="status"></span></div>
<h2>Review ({len(review)}): disagrees with the current label, or p99 between {band[0]} and {band[1]}</h2>
<div class="grid">{''.join(card(p) for p in review)}</div>
<details><summary>Everything else ({len(rest)}), for spot checks</summary>
<div class="grid">{''.join(card(p) for p in rest)}</div></details>
<script>
const KEY = "grayscale-review-labels";
let saved = {{}};
try {{ saved = JSON.parse(localStorage.getItem(KEY) || "{{}}"); }} catch (e) {{}}
document.querySelectorAll(".card").forEach(card => {{
  const id = card.dataset.id, value = saved[id] || card.dataset.default;
  card.querySelector(`input[value="${{value}}"]`).checked = true;
  card.classList.toggle("changed", value !== card.dataset.default);
  card.querySelectorAll("input").forEach(input => input.addEventListener("change", () => {{
    saved[id] = input.value;
    card.classList.toggle("changed", input.value !== card.dataset.default);
    try {{ localStorage.setItem(KEY, JSON.stringify(saved)); }} catch (e) {{}}
  }}));
}});
document.getElementById("copy").addEventListener("click", async () => {{
  const labels = {{}};
  document.querySelectorAll(".card").forEach(card => {{
    labels[card.dataset.id] = card.querySelector("input:checked").value;
  }});
  const text = JSON.stringify(labels);
  try {{ await navigator.clipboard.writeText(text); document.getElementById("status").textContent = "Copied " + Object.keys(labels).length + " labels"; }}
  catch (e) {{ prompt("Copy these labels", text); }}
}});
</script></body></html>"""
    with open(out, "w") as f:
        f.write(page)


def main() -> None:
    parser = argparse.ArgumentParser(description="Calibrate the grayscale threshold")
    parser.add_argument("--work-dir", default="grayscale-calibration")
    parser.add_argument("--out", default="grayscale-review.html")
    parser.add_argument("--per-class", type=int, default=1000)
    parser.add_argument("--bw", type=int, default=300)
    parser.add_argument("--hard", type=int, default=100)
    parser.add_argument("--band", type=int, nargs=2, default=[4, 40])
    parser.add_argument("--preset", type=int, help="threshold used to preset the review cards")
    parser.add_argument("--labels", help="JSON of user labels {id: bw|color} to score thresholds against")
    args = parser.parse_args()

    cache_dir = os.path.join(args.work_dir, "images")
    os.makedirs(cache_dir, exist_ok=True)
    data_path = os.path.join(args.work_dir, "samples.json")

    if os.path.exists(data_path):
        with open(data_path) as f:
            posts = json.load(f)
    else:
        posts = sample(args.per_class, args.bw, args.hard)
        with ThreadPoolExecutor(max_workers=8) as pool:
            posts = list(pool.map(lambda p: measure(p, cache_dir), posts))
        with open(data_path, "w") as f:
            json.dump(posts, f)

    if args.labels:
        with open(args.labels) as f:
            labels = {int(k): v == "bw" for k, v in json.load(f).items()}
        labelled = [dict(p, truth=labels[p["id"]]) for p in posts if p["id"] in labels]
        t, errors, scored = best_threshold(labelled, "truth")
        print(f"user labels: {len(labelled)}, best threshold p99 <= {t}, errors {errors}")
        for e, th in scored:
            if abs(th - t) <= 10:
                print(f"  threshold {th:>2}: errors {e}")
        for p in labelled:
            if (p["p99"] <= t) != p["truth"]:
                print(f"  wrong at {t}: #{p['id']} p99 {p['p99']:.0f} truth {'bw' if p['truth'] else 'color'}")
        return

    t, errors, _ = best_threshold(posts, "db_grayscale")
    counts = {g: sum(p["group"] == g for p in posts) for g in ("bw", "color", "muted")}
    print(f"samples {len(posts)} {counts}, threshold vs db labels: p99 <= {t}, disagreements {errors}")
    render(posts, args.preset if args.preset is not None else t, tuple(args.band), args.out)
    print(f"wrote {args.out}")


if __name__ == "__main__":
    main()
