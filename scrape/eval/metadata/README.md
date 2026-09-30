# Metadata eval

A labeled set of about 300 posts and a scorer, so every change to camera, film and lens
extraction has numbers behind it. Offline only: the scripts read the public API, the S3
comments cache and Reddit, and never write to any of them.

Run everything from `scrape/`. The scripts read `scrape/.env` (OpenRouter, Reddit, AWS).

## Files

Everything in `data/` is local and not committed. Rebuild it with the steps below.

| File | What |
|---|---|
| `data/sample.json` | the posts: text, OP comments, image, stored metadata, stratum, set |
| `data/catalog.json` | catalog snapshot the labels were made against |
| `data/drafts.json` | draft labels from a strong model, the starting point for review |
| `data/gold.json` | reviewed labels, the truth for scoring |
| `data/review.html` | the labeling page |
| `data/runs/<tag>.json` | cached predictions of one scored run |
| `data/reports/<tag>.md`, `.html`, `.json` | score reports |
| `normalize_cases.json` | shared normalization fixture (Python and Go) |

## 1. Sample

```
uv run python -m eval.metadata.sample --seed 2026
```

Pages through `/v1/posts?sort=random&seed=2026` and gives each post the first stratum it fits
that still has room:

| Stratum | Count | Rule |
|---|---|---|
| `negative` | 40 | no `[` in the title and no make, film name or letter-digit model name |
| `not_in_catalog` | 40 | a camera make followed by a model-like word that isn't a catalog model as written (includes aliases like `AE1P`, `N80`, `6x7` and real gaps like `C33`) |
| `ids_37k` | 30 | ids 37000 to 37999, the failed-batch window |
| `ids_38k_meta` | 90 | ids 38000 and up with stored metadata, measures today's precision |
| `pre_2025` | 100 | posted before 2025, the backfill target |
| `fewshot` | 15 | extra posts kept out of scoring, for prompt examples |

OP comments come from the S3 cache (`analog-comments/<id>.json`), or Reddit when missing. Only
the post author's own comments are kept, oldest first, 1,500 characters in total.

The API's random order shifts as posts are added, so the same seed gives a different sample
later. Keep `sample.json` and `gold.json` together: labels only match the sample they came from.

## 2. Draft labels

```
uv run python -m eval.metadata.prelabel --model anthropic/claude-fable-5.1 --max-cost 5
```

Ten posts per call with the catalog in a cached system prompt. Resumes where it stopped.
Values outside the catalog are moved to `unmatched`. About $4 for the full set with Fable 5.1.

## 3. Review

```
uv run python -m eval.metadata.review
open eval/metadata/data/review.html
```

Each card shows the stored value (what the site has), the draft, and an editable label. Fix it,
tick Reviewed. Useful filters: `Stored ≠ draft` for the cards that need a look, and
`Stored = draft, not reviewed` with **Mark shown as reviewed** for the easy ones. Choices save in
the browser. **Download** saves `gold.json`: move it to `data/gold.json`. **Import** loads a
`gold.json` back into the page (for another browser, or after a rebuild).

The label rules are on the page and in `prelabel.py` (`RULES`).

## 4. Score

```
uv run python -m eval.metadata.score --extractor stored                  # what the site has
uv run python -m eval.metadata.score --extractor current --tag baseline  # today's extractor, fresh
uv run python -m eval.metadata.score --extractor current --model google/gemini-2.5-flash --tag baseline-flash
uv run python -m eval.metadata.score --tag baseline --rescore            # re-score cached predictions
uv run python -m eval.metadata.score --extractor v2 --tag v2 --compare baseline
```

- Scores the reviewed `test` posts against `data/catalog.json`, never the live catalog.
- `--gold drafts` scores against the unreviewed drafts (a smoke test before review is done).
- `--model` defaults to `OPENROUTER_MODEL` from `.env`.
- Predictions are cached in `data/runs/<tag>.json`, so `--rescore` costs nothing, for example
  after fixing a gold label.
- `--compare <tag>` adds change columns to the tables and the other run's values to each
  disagreement card.

Per field (`camera` and `film` are the make and model or type together):
- **precision:** of the values set, the share equal to gold
- **recall:** of the gold values, the share set correctly
- **wrong:** set and different from gold, where the post names that camera or film. This
  includes a model stored for a camera that isn't in the catalog (G2 stored as `g1`). The worst
  error: a confident wrong label on the site.
- **spurious:** set on a post that names no camera or film at all
- **unmatched recall:** of the gold "not in catalog" mentions, the share the extractor reported.
  Only for extractors that report them.

The HTML report lists every disagreement as a card, wrong values first.

## Tests

```
uv run pytest eval
```
