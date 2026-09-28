.PHONY: up upd down build db infra mon log test bench-db bench-migrate bench-seed bench

up :
	docker compose up

upd :
	docker compose up -d

down :
	docker compose down --remove-orphans

build :
	docker compose up -d --build

db :
	docker compose up -d postgres

infra :
	docker compose up -d postgres weaviate i2v-neural

log :
	docker compose logs --follow

test :
	go test ./...

BENCH_DSN ?= postgres://postgres:postgres@localhost:55432/analogdb?sslmode=disable
BENCH_POSTS ?= 100000
BENCH_ARGS ?= -benchtime=2s
BENCH_EXPLAIN ?=
PSQL ?= docker exec -i analogdb-bench psql -U postgres -d analogdb

bench-db :
	docker run -d --name analogdb-bench -e POSTGRES_PASSWORD=postgres -e POSTGRES_DB=analogdb -p 55432:5432 postgres:15

bench-migrate :
	ANALOGDB_BENCH_DSN="$(BENCH_DSN)" ANALOGDB_BENCH_MIGRATE=1 go test ./postgres -run '^TestBenchMigrate$$' -count=1

bench-seed : bench-migrate
	$(PSQL) -v posts=$(BENCH_POSTS) < bench/seed.sql

bench :
	ANALOGDB_BENCH_DSN="$(BENCH_DSN)" ANALOGDB_BENCH_EXPLAIN="$(if $(BENCH_EXPLAIN),$(abspath $(BENCH_EXPLAIN)))" go test ./postgres -run '^$$' -bench '^BenchmarkFindPosts$$' $(BENCH_ARGS)
