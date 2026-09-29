"""Build the static labeling page, data/review.html.

uv run python -m eval.metadata.review
"""

import html
import json

from .common import CATALOG_PATH, DATA, DRAFTS_PATH, GOLD_PATH, SAMPLE_PATH, read_json
from .prelabel import RULES

REVIEW_PATH = DATA / "review.html"

PAGE = """<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Metadata Labels</title>
<style>
:root { --bg:#fafafa; --fg:#1a1a1a; --card:#fff; --muted:#666; --line:#ddd; --accent:#2563eb;
  --warn:#b45309; --warn-bg:#fef3c7; --bad:#dc2626; --on-accent:#fff; }
@media (prefers-color-scheme: dark) { :root { --bg:#111; --fg:#eee; --card:#1c1c1c; --muted:#999;
  --line:#333; --accent:#60a5fa; --warn:#fbbf24; --warn-bg:#3a2a05; --bad:#f87171; --on-accent:#0b1220; } }
* { box-sizing:border-box; }
body { background:var(--bg); color:var(--fg); font:14px/1.4 system-ui,sans-serif; margin:0; padding:16px; }
h1 { font-size:20px; margin:0 0 4px; }
p, .rules { color:var(--muted); margin:4px 0 12px; max-width:80ch; }
.rules { white-space:pre-wrap; font-size:13px; }
.bar { position:sticky; top:0; background:var(--bg); padding:8px 0; border-bottom:1px solid var(--line);
  z-index:2; display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
a { color:var(--accent); }
button, .file { background:var(--accent); color:var(--on-accent); border:0; border-radius:6px; padding:7px 12px;
  font:inherit; cursor:pointer; }
.file input { display:none; }
select, input, textarea { font:inherit; color:var(--fg); background:var(--bg); border:1px solid var(--line);
  border-radius:4px; padding:4px 6px; width:100%; }
select { width:auto; }
input.invalid { border-color:var(--bad); outline:1px solid var(--bad); }
.list { display:grid; gap:16px; max-width:900px; margin-top:12px; }
.card { background:var(--card); border:1px solid var(--line); border-radius:8px; padding:12px; }
.card.done { opacity:.6; }
.card.done:focus-within, .card.done:hover { opacity:1; }
.head { display:flex; gap:8px; flex-wrap:wrap; align-items:center; color:var(--muted); font-size:12px; margin-bottom:8px; }
.head b { color:var(--fg); font-size:14px; }
.badge { border:1px solid var(--line); border-radius:10px; padding:0 8px; }
.badge.differs { color:var(--warn); border-color:var(--warn); }
.top { display:grid; grid-template-columns:200px 1fr; gap:12px; }
.top img { width:100%; border-radius:4px; display:block; }
.title { font-weight:600; margin-bottom:6px; overflow-wrap:anywhere; }
details { color:var(--muted); font-size:13px; margin:4px 0; overflow-wrap:anywhere; }
summary { cursor:pointer; }
.dnote { color:var(--warn); font-size:13px; }
table { width:100%; border-collapse:collapse; margin-top:10px; table-layout:fixed; }
th, td { text-align:left; padding:3px 4px; border-bottom:1px solid var(--line); vertical-align:middle;
  overflow-wrap:anywhere; }
th { color:var(--muted); font-weight:500; font-size:12px; }
th:nth-child(1) { width:22%; } th:nth-child(2), th:nth-child(3) { width:22%; }
td.diff { background:var(--warn-bg); }
td.same { color:var(--muted); }
.extra { display:grid; grid-template-columns:1fr 1fr; gap:8px; margin-top:8px; }
.extra label, .notes label { font-size:12px; color:var(--muted); }
.notes { margin-top:8px; }
textarea { height:34px; resize:vertical; }
.done-row { margin-top:8px; display:flex; gap:12px; align-items:center; }
.done-row label { display:flex; gap:6px; align-items:center; cursor:pointer; }
.done-row input { width:auto; }
@media (max-width: 600px) {
  body { padding:8px 16px; }
  .top { grid-template-columns:1fr; }
  .top img { max-height:260px; object-fit:cover; }
  .extra { grid-template-columns:1fr; }
  th:nth-child(1) { width:26%; }
}
</style></head><body>
<h1>Metadata labels</h1>
<p>Each card is prefilled from the draft (or your earlier labels). Fix it, tick <b>Reviewed</b>, move on.
The <i>stored</i> column is what the site has today. Amber cells are where the stored value and the draft disagree.
A red field isn't in the catalog. Choices are saved in this browser. When done, press <b>Download</b>
and save it as <code>data/gold.json</code>.</p>
<details><summary>Label rules</summary><div class="rules">__RULES__</div></details>
<div class="bar">
  <select id="filter"></select>
  <span id="progress"></span>
  <button id="bulk">Mark shown as reviewed</button>
  <button id="copy">Copy labels</button>
  <button id="download">Download</button>
  <label class="file">Import<input type="file" id="import" accept="application/json"></label>
  <span id="status"></span>
</div>
<datalist id="dl-cmake"></datalist><datalist id="dl-fmake"></datalist><datalist id="dl-speed"></datalist>
<div class="list" id="list"></div>
<script>
const D = __DATA__;
const KEY = "metadata-review-v1";
const FIELDS = ["camera_make","camera_model","film_make","film_type","film_speed","focal_length","aperture"];
const NAMES = {camera_make:"Camera make",camera_model:"Camera model",film_make:"Film make",
  film_type:"Film type",film_speed:"Film speed",focal_length:"Focal length",aperture:"Aperture"};
const SPEEDS = [1,2,3,6,12,20,25,50,64,80,100,125,160,200,250,320,400,500,800,1000,1600,3200,6400];
let saved = {};
try { saved = JSON.parse(localStorage.getItem(KEY) || "{}"); } catch (e) {}
function persist() { try { localStorage.setItem(KEY, JSON.stringify(saved)); } catch (e) {} }

const esc = s => String(s ?? "").replace(/[&<>"']/g, c => ({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));
const norm = v => v === null || v === undefined || v === "" ? "" : String(v).trim().toLowerCase();
const camMakes = [...new Set(D.catalog.cameras.map(c => c.make))].sort();
const filmMakes = [...new Set(D.catalog.films.map(f => f.make))].sort();
const camModels = m => D.catalog.cameras.filter(c => c.make === m).map(c => c.model).sort();
const filmTypes = m => D.catalog.films.filter(f => f.make === m);
const options = values => values.map(v => `<option value="${esc(v)}">`).join("");
const fill = (id, values) => { document.getElementById(id).innerHTML = options(values); };
fill("dl-cmake", camMakes); fill("dl-fmake", filmMakes); fill("dl-speed", SPEEDS);

function unmatchedText(list, kind) {
  return (list || []).filter(u => u.kind === kind).map(u => u.raw).join("; ");
}
function fromLabel(src, isGold) {
  const l = {};
  FIELDS.forEach(f => l[f] = src[f] ?? "");
  l.camera_unmatched = unmatchedText(src.unmatched, "camera");
  l.film_unmatched = unmatchedText(src.unmatched, "film");
  l.note = isGold ? (src.note || "") : "";
  l.reviewed = !!(isGold && src.reviewed);
  return l;
}
const initial = p => D.gold[p.id] ? fromLabel(D.gold[p.id], true) : fromLabel(D.drafts[p.id] || {}, false);
const label = p => saved[p.id] || initial(p);
function differs(p) {
  const d = D.drafts[p.id];
  return !!d && FIELDS.some(f => norm(p.stored[f]) !== norm(d[f]));
}
function valid(l, f) {
  const v = norm(l[f]);
  if (!v) return true;
  if (f === "camera_make") return camMakes.includes(v);
  if (f === "camera_model") return camModels(norm(l.camera_make)).includes(v);
  if (f === "film_make") return filmMakes.includes(v);
  if (f === "film_type") return filmTypes(norm(l.film_make)).some(x => x.type === v);
  if (f === "film_speed") return SPEEDS.includes(+v);
  if (f === "focal_length") return /^\\d+$/.test(v) && +v >= 8 && +v <= 800;
  if (f === "aperture") return /^f\\/\\d+(\\.\\d+)?$/.test(v);
  return true;
}
function listFor(p, f, l) {
  if (f === "camera_make") return "dl-cmake";
  if (f === "film_make") return "dl-fmake";
  if (f === "film_speed") return "dl-speed";
  if (f === "camera_model" || f === "film_type") return `dl-${f}-${p.id}`;
  return "";
}
function refreshLists(card, p, l) {
  card.querySelector(`#dl-camera_model-${p.id}`).innerHTML = options(camModels(norm(l.camera_make)));
  card.querySelector(`#dl-film_type-${p.id}`).innerHTML = options(filmTypes(norm(l.film_make)).map(x => x.type));
  card.querySelectorAll("input[data-f]").forEach(i => {
    if (FIELDS.includes(i.dataset.f)) i.classList.toggle("invalid", !valid(l, i.dataset.f));
  });
}

function render(p) {
  const l = label(p), d = D.drafts[p.id] || {};
  const date = new Date(p.time * 1000).toISOString().slice(0, 10);
  const rows = FIELDS.map(f => {
    const s = p.stored[f] ?? "", dv = d[f] ?? "";
    const cls = norm(s) === norm(dv) ? "same" : "diff";
    const list = listFor(p, f, l);
    return `<tr><td>${NAMES[f]}</td><td class="${cls}">${esc(s)}</td><td class="${cls}">${esc(dv)}</td>
      <td><input data-f="${f}" value="${esc(l[f])}" ${list ? `list="${list}"` : ""} autocomplete="off"></td></tr>`;
  }).join("");
  const comments = p.op_comments.length
    ? `<details><summary>OP comments (${p.op_comments.length})</summary>${p.op_comments.map(c => `<div>• ${esc(c)}</div>`).join("")}</details>` : "";
  const desc = p.description ? `<details open><summary>Description</summary>${esc(p.description)}</details>` : "";
  const card = document.createElement("div");
  card.className = "card";
  card.dataset.id = p.id;
  card.innerHTML = `
    <div class="head"><b>#${p.id}</b><span class="badge">${esc(p.stratum)}</span><span>${date}</span>
      <a href="https://analogdb.com/post/${p.id}" target="_blank" rel="noopener">analogdb</a>
      <a href="${esc(p.permalink)}" target="_blank" rel="noopener">reddit</a>
      ${differs(p) ? '<span class="badge differs">stored ≠ draft</span>' : ""}</div>
    <div class="top">
      <a href="${esc(p.image)}" target="_blank" rel="noopener"><img loading="lazy" src="${esc(p.image)}" alt=""></a>
      <div><div class="title">${esc(p.title)}</div>${desc}${comments}
        ${d.note ? `<div class="dnote">Draft note: ${esc(d.note)}</div>` : ""}</div>
    </div>
    <table><thead><tr><th>Field</th><th>Stored</th><th>Draft</th><th>Label</th></tr></thead><tbody>${rows}</tbody></table>
    <datalist id="dl-camera_model-${p.id}"></datalist><datalist id="dl-film_type-${p.id}"></datalist>
    <div class="extra">
      <div><label>Camera not in catalog</label><input data-f="camera_unmatched" value="${esc(l.camera_unmatched)}" placeholder="as written, ; between several"></div>
      <div><label>Film not in catalog</label><input data-f="film_unmatched" value="${esc(l.film_unmatched)}" placeholder="as written, ; between several"></div>
    </div>
    <div class="notes"><label>Note</label><textarea data-f="note">${esc(l.note)}</textarea></div>
    <div class="done-row"><label><input type="checkbox" data-f="reviewed" ${l.reviewed ? "checked" : ""}> Reviewed</label></div>`;
  card.classList.toggle("done", l.reviewed);
  refreshLists(card, p, l);
  card.addEventListener("input", e => {
    const f = e.target.dataset.f;
    if (!f) return;
    const cur = {...label(p)};
    cur[f] = e.target.type === "checkbox" ? e.target.checked : e.target.value;
    if (f === "film_type" || f === "film_make") {
      const match = filmTypes(norm(cur.film_make)).find(x => x.type === norm(cur.film_type));
      if (match) { cur.film_speed = String(match.speed); card.querySelector('[data-f="film_speed"]').value = match.speed; }
    }
    saved[p.id] = cur;
    persist();
    card.classList.toggle("done", !!cur.reviewed);
    refreshLists(card, p, cur);
    progress();
  });
  return card;
}

const list = document.getElementById("list");
const cards = D.posts.map(p => { const c = render(p); list.appendChild(c); return [p, c]; });

const filter = document.getElementById("filter");
const strata = [...new Set(D.posts.map(p => p.stratum))];
filter.innerHTML = [["all","All"],["todo","Not reviewed"],["differs","Stored ≠ draft"],["agrees","Stored = draft, not reviewed"],["notes","Has draft note"]]
  .concat(strata.map(s => ["s:" + s, "Stratum: " + s]))
  .map(([v, t]) => `<option value="${v}">${t}</option>`).join("");
function applyFilter() {
  const v = filter.value;
  cards.forEach(([p, c]) => {
    const l = label(p);
    const show = v === "all" || (v === "todo" && !l.reviewed) || (v === "differs" && differs(p))
      || (v === "agrees" && !l.reviewed && D.drafts[p.id] && !differs(p))
      || (v === "notes" && (D.drafts[p.id] || {}).note) || v === "s:" + p.stratum;
    c.style.display = show ? "" : "none";
  });
}
filter.addEventListener("change", applyFilter);

document.getElementById("bulk").addEventListener("click", () => {
  const shown = cards.filter(([p, c]) => c.style.display !== "none" && !label(p).reviewed);
  if (!shown.length || !confirm(`Mark ${shown.length} shown cards as reviewed?`)) return;
  shown.forEach(([p, c]) => {
    saved[p.id] = {...label(p), reviewed: true};
    c.querySelector('[data-f="reviewed"]').checked = true;
    c.classList.add("done");
  });
  persist();
  progress();
});

function progress() {
  const n = D.posts.filter(p => label(p).reviewed).length;
  document.getElementById("progress").textContent = `${n} / ${D.posts.length} reviewed`;
}
progress();

const toInt = v => { const n = parseInt(v, 10); return Number.isFinite(n) ? n : null; };
function exportGold() {
  const labels = D.posts.map(p => {
    const l = label(p), out = {post_id: p.id, set: p.set, stratum: p.stratum, reviewed: !!l.reviewed};
    FIELDS.forEach(f => { const v = norm(l[f]); out[f] = v ? v : null; });
    out.film_speed = toInt(out.film_speed); out.focal_length = toInt(out.focal_length);
    out.unmatched = [];
    ["camera", "film"].forEach(k => String(l[k + "_unmatched"] || "").split(";").map(s => s.trim())
      .filter(Boolean).forEach(raw => out.unmatched.push({kind: k, raw})));
    out.note = l.note || "";
    return out;
  });
  return {catalog_fetched: D.catalog.fetched, reviewed: labels.filter(l => l.reviewed).length, labels};
}
const status = t => { document.getElementById("status").textContent = t; };
document.getElementById("copy").addEventListener("click", async () => {
  const text = JSON.stringify(exportGold(), null, 1);
  try { await navigator.clipboard.writeText(text); status("Copied"); } catch (e) { prompt("Copy these labels", text); }
});
document.getElementById("download").addEventListener("click", () => {
  const blob = new Blob([JSON.stringify(exportGold(), null, 1) + "\\n"], {type: "application/json"});
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob); a.download = "gold.json"; a.click();
  URL.revokeObjectURL(a.href);
});
document.getElementById("import").addEventListener("change", async e => {
  const file = e.target.files[0];
  if (!file) return;
  try {
    const data = JSON.parse(await file.text());
    (data.labels || []).forEach(g => { saved[g.post_id] = fromLabel(g, true); });
    persist();
    location.reload();
  } catch (err) { status("Import failed: " + err.message); }
});
</script></body></html>
"""


def build() -> str:
    sample = read_json(SAMPLE_PATH)
    catalog = read_json(CATALOG_PATH)
    drafts = read_json(DRAFTS_PATH)["labels"] if DRAFTS_PATH.exists() else {}
    gold = {}
    if GOLD_PATH.exists():
        gold = {str(g["post_id"]): g for g in read_json(GOLD_PATH)["labels"]}
    data = {
        "posts": sample["posts"],
        "drafts": drafts,
        "gold": gold,
        "catalog": catalog,
    }
    blob = json.dumps(data, ensure_ascii=False).replace("</", "<\\/")
    return PAGE.replace("__RULES__", html.escape(RULES)).replace("__DATA__", blob)


def main():
    REVIEW_PATH.write_text(build())
    print(f"wrote {REVIEW_PATH}")


if __name__ == "__main__":
    main()
