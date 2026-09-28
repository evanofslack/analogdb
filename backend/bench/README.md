# Post query benchmarks

Tooling to measure the post queries against a realistic, deterministic dataset.

- `seed.sql` fills a migrated database with synthetic posts. The default is 100k posts
  (production has about 23k). Times span about 5 years, scores are skewed with many ties,
  there are about 10k authors, and camera and film values come from `scrape/data/*.json`.
  Each post has 5 colors and about 10 keywords drawn from a skewed vocabulary. Titles come
  from a word list and some contain commas. The same row count always produces the same data.
- `../postgres/bench_test.go` holds `BenchmarkFindPosts`, which calls the real services.
  It is skipped unless `ANALOGDB_BENCH_DSN` is set.

## Running

From `backend/`:

```bash
make bench-db      # postgres:15 in docker, published on localhost:55432
make bench-seed    # run migrations, then seed.sql (about 20s for 100k posts)
make bench         # run the benchmark
```

`make bench-seed` removes all existing rows. Options:

| Variable | Default | Meaning |
|---|---|---|
| `BENCH_DSN` | `postgres://postgres:postgres@localhost:55432/analogdb?sslmode=disable` | Database to use |
| `BENCH_POSTS` | `100000` | Number of posts to seed |
| `BENCH_ARGS` | `-benchtime=2s` | Extra `go test` flags, for example `-count=3` |
| `BENCH_EXPLAIN` | empty | File to write `EXPLAIN (ANALYZE, BUFFERS)` for every statement of every case |
| `PSQL` | `docker exec -i analogdb-bench psql -U postgres -d analogdb` | psql command used by `bench-seed`. With a local psql use `PSQL='psql "$(BENCH_DSN)"'` |

```bash
make bench BENCH_EXPLAIN=bench/explain.txt
make bench-migrate   # apply new migrations to an already seeded database
```

### Read only mode

Set `ANALOGDB_BENCH_READONLY=1` to connect with `default_transaction_read_only=on`. The
benchmark then cannot write anything, so it can run against any database, including a
restored dump:

```bash
ANALOGDB_BENCH_READONLY=1 make bench BENCH_DSN='postgres://user:pass@host:5432/analogdb'
```

## Cases

| Case | Request it models |
|---|---|
| `posts_default` | `/posts` (time sort, 20 rows) |
| `sort_score_page5` | `/posts?sort=score`, fifth page |
| `sort_random` | `/posts?sort=random&seed=37` |
| `sort_random_mod_sql` | page select only, `ORDER BY MOD(time, 37), time DESC` |
| `sort_random_md5_sql` | page select only, `ORDER BY md5(id::text \|\| '42'), id` |
| `keyword_portrait` | `/posts?keyword=portrait` |
| `color_teal_min_0.2` | `/posts?color=teal&min_color=0.2` |
| `camera_nikon_fm2` | `/posts?camera_make=nikon&camera_model=fm2` |
| `title_beach` | `/posts?title=beach` |
| `cameras_with_counts` | `/cameras?include_counts=true&exclude_zero_counts=true` |

The color case uses `teal` because the scraper maps `blue` to `teal`, so no real post has
the html color `blue`.

## Results

100k seeded posts, `postgres:15` in docker (colima VM with 2 CPUs and 2 GB) on an Apple M5,
`-benchtime=2s -count=3`, median of the three runs. Before is the code and schema before
migration `000009_add_indexes`. After is the same seeded database after `make bench-migrate`.

| Case | Before (ms/op) | After (ms/op) | Plan after |
|---|---:|---:|---|
| `posts_default` | 836 | 3.7 | yes: `idx_pictures_time_id`, colors and keywords by `idx_*_post_id` |
| `sort_score_page5` | 737 | 4.5 | yes: `idx_pictures_score_id` for page and count |
| `sort_random` | 808 | 22.0 | seq scan and sort of `pictures` (no index by design), colors and keywords by `idx_*_post_id` |
| `sort_random_mod_sql` | 26.7 | 19.4 | seq scan and sort (no index by design) |
| `sort_random_md5_sql` | 48.3 | 39.9 | seq scan and sort (no index by design) |
| `keyword_portrait` | 872 | 5.3 | yes: `idx_keywords_word` for page and count |
| `color_teal_min_0.2` | 855 | 10.7 | yes: `idx_colors_html` for page and count |
| `camera_nikon_fm2` | 734 | 1.8 | yes: `idx_pictures_camera` |
| `title_beach` | 764 | 45.1 | page walks `idx_pictures_time_id`, count is a seq scan (no trigram index) |
| `cameras_with_counts` | 18.1 | 4.5 | yes: index only scan of `idx_pictures_camera` |

Before, every post query aggregated all of `colors` and `keywords` (seq scans and on disk
sorts), which is where the ~800 ms went. The two raw random cases run the same SQL before and
after, so their difference is run to run noise. They show `md5(id::text || '42')` costs about
twice `MOD(time, seed)` at 100k rows. `title_beach` is dominated by the count, a full scan with
`ILIKE`: about 42 ms at 100k rows, so roughly 10 ms at production size.
