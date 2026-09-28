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
