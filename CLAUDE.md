# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AnalogDB is a full-stack application for managing and discovering analog photography collections. It consists of a Go backend API, Next.js frontend, Python scraping services, and analytics infrastructure.

## Architecture

- **Backend (`/backend/`)**: Go HTTP API using chi router, PostgreSQL for data, Redis for caching, Weaviate for vector similarity
- **Frontend (`/web/`)**: Next.js React application with TypeScript, CSS modules, Mantine components
- **Scraping (`/scrape/`)**: Python services for data ingestion and ETL pipelines using Dagster
- **Consumer (`/consumer/`)**: Go service for processing analytics events with Kafka and ClickHouse
- **Infrastructure (`/infra/`)**: Docker compose for observability stack (Prometheus, Grafana, Loki, Tempo)
- **API Clients (`/api/clients/`)**: Auto-generated TypeScript and Python clients from OpenAPI spec
- **Bench (`/backend/bench/`)**: Synthetic seed data and post query benchmarks (see its README)

## Common Commands

Tasks run through [just](https://github.com/casey/just). The root `justfile` loads one module per
service, so `just backend test` from the root is the same as `just test` inside `/backend/`.
Run `just` to list every recipe.

### Whole Repo
```bash
just setup                 # Create the docker network and install deps for every service
just up                    # Start backend, consumer, web and infra containers
just down                  # Stop them
just ps                    # List analogdb containers
just test                  # Backend, consumer and scrape tests
just lint                  # go vet, ruff and next lint
just fmt                   # gofmt, ruff and black, prettier
just check                 # lint, test and swagger-check
just clean                 # Remove build and test artifacts
just nuke                  # Delete all containers, volumes and the network (asks first)
```

### Backend
```bash
just backend up            # Start all backend containers
just backend infra         # Start just PostgreSQL, Weaviate, and i2v-neural
just backend db            # Start just PostgreSQL
just backend run           # Run the API on the host
just backend test          # go test -race, sets colima testcontainers env when present
just backend psql          # psql shell in the postgres container
just backend logs postgres # Follow logs, all services when none given
just backend bench         # Post query benchmark, see /backend/bench/README.md
```

### Frontend
```bash
just web dev               # Start development server
just web build             # Build for production
just web lint              # Run ESLint
just web rebuild           # Rebuild and start the web container
```

### Python Services
```bash
just scrape sync           # Install dependencies
just scrape dev            # Run the Dagster dev server
just scrape test           # Run pytest over src and packages
```

### Consumer and Infra
```bash
just consumer test         # Consumer Go tests
just consumer clickhouse   # clickhouse-client in the clickhouse container
just infra up              # Start Prometheus, Grafana, Loki, Tempo
```

### API Client Generation
```bash
just swagger               # Generate OpenAPI spec from Go code
just swagger-check         # Fail if the committed spec is stale, as CI does
just gen-client-python     # Generate Python client
just gen-client-typescript # Generate TypeScript client
just gen-clients           # Both clients

# These commands:
# 1. Generate swagger.json/yaml from Go annotations in backend
# 2. Use openapi-generator-cli to create clients
# 3. Update package dependencies where needed
```

### Protocol Buffers
```bash
just proto                 # Generate analytics event code for backend and consumer
```

### Load Testing
```bash
just load scripts/<name>.js  # Run a k6 script from /test/k6/
```

## Code Architecture Notes

### Backend Structure
- **Domain models**: `/backend/*.go` files define core types (Post, Camera, Film, Author, etc.)
- **HTTP handlers**: `/backend/server/` contains REST API implementation
- **Data layer**: `/backend/postgres/` for primary storage, `/backend/redis/` for caching
- **Vector operations**: `/backend/weaviate/` handles image similarity using embeddings
- **Configuration**: `/backend/config/` centralizes all app configuration
- **Observability**: `/backend/logger/` (slog), `/backend/metrics/` (Prometheus), `/backend/tracer/` (OpenTelemetry)
- **Analytics events**: `/backend/events/` publishes to Kafka; generated proto code lives in `/backend/internal/gen/proto/`
- **Entrypoint**: `/backend/cmd/analogdb/`

Note: the backend uses `log/slog` for logging, wrapped in `logger.Logger` (`/backend/logger/logger.go`), not zerolog.

### Frontend Structure
- **Pages**: `/web/app/` contains Next.js App Router routes (about, admin, post, actions, docs)
- **Components**: `/web/components/` has reusable React components with co-located CSS modules
- **API Integration**: Uses generated TypeScript client from `/api/clients/typescript/`
- **State Management**: Custom hooks in `/web/hooks/` for data fetching (usePosts, useCameras, useFilms)
- **Styling**: CSS Modules pattern with Mantine component library

### Data Pipeline
- **Ingestion**: Python scrapers collect data from external sources
- **Processing**: Dagster orchestrates ETL workflows in `/scrape/`
- **Storage**: PostgreSQL for relational data, S3 for images, Weaviate for embeddings
- **Analytics**: Events flow through Kafka to ClickHouse via the consumer service

### API Design
- **REST endpoints**: Follow conventional patterns (/posts, /cameras, /films, /authors)
- **OpenAPI spec**: Auto-generated from Go struct annotations via swag
- **Authentication**: Basic auth for admin endpoints
- **Pagination**: Cursor-based pagination for large result sets

## Development Workflow

1. **Backend changes**: Modify Go code, run tests with `just backend test`, update swagger with `just swagger`
2. **Frontend changes**: Work in `/web/`, use `npm run dev` for hot reload
3. **API changes**: Regenerate clients with `just gen-clients` after OpenAPI spec updates
4. **Database changes**: Add migrations to `/backend/postgres/migrations/`
5. **Infrastructure**: Use docker-compose files for consistent development environments

## Testing

- **Backend**: `just backend test` runs all unit tests
- **Scrape**: `just scrape test` runs pytest
- **Frontend**: No test suite yet, `just web lint` runs ESLint
- **Integration**: Docker compose in `/test/k6/` for load testing
- **Database**: Test containers used in Go tests for isolated database testing
