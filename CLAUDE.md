# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AnalogDB is a full-stack application for managing and discovering analog photography collections. It consists of a Go backend API, Next.js frontend, Python scraping services, and analytics infrastructure.

## Architecture

- **Backend (`/backend/`)**: Go HTTP API using chi router, PostgreSQL for data, Redis for caching, Weaviate for vector similarity
- **Frontend (`/web/`)**: Next.js React application with TypeScript, CSS modules, Mantine components
- **Scraping (`/scraper/`, `/scrape/`)**: Python services for data ingestion and ETL pipelines using Dagster
- **Consumer (`/consumer/`)**: Go service for processing analytics events with Kafka and ClickHouse
- **Infrastructure (`/infra/`)**: Docker compose for observability stack (Prometheus, Grafana, Loki, Tempo)
- **API Clients (`/api/clients/`)**: Auto-generated TypeScript and Python clients from OpenAPI spec

## Common Commands

### Backend Development
```bash
# Start backend services (from /backend/)
make upd                    # Start with docker-compose in background
make infra                  # Start just PostgreSQL, Weaviate, and i2v-neural
make test                   # Run all Go tests
go test ./...              # Alternative test command

# Database operations
make db                     # Start just PostgreSQL
```

### Frontend Development
```bash
# Frontend (from /web/)
npm run dev                 # Start development server
npm run build              # Build for production
npm run lint               # Run ESLint

# Install dependencies after API client changes
npm install                # May be needed after client regeneration
```

### Python Services
```bash
# Scraping service (from /scrape/)
uv sync                    # Install dependencies

# Legacy scraper (from /scraper/)
# Uses Pipfile/Pipenv for dependency management
```

### API Client Generation
```bash
# Root level commands for API client generation
make swagger               # Generate OpenAPI spec from Go code
make gen-client-python     # Generate Python client
make gen-client-typescript # Generate TypeScript client

# These commands:
# 1. Generate swagger.json/yaml from Go annotations in backend
# 2. Use openapi-generator-cli to create clients
# 3. Update package dependencies where needed
```

### Protocol Buffers
```bash
# Generate protobuf code for analytics events
make proto                 # Generate for both backend and consumer
make proto-backend         # Generate just for backend
make proto-consumer        # Generate just for consumer
```

### Infrastructure & Testing
```bash
# Infrastructure
docker-compose -f docker-compose-dev.yaml up  # Development stack

# Load testing (from /test/k6/)
./run.sh                   # Run k6 performance tests
```

## Code Architecture Notes

### Backend Structure
- **Domain models**: `/backend/*.go` files define core types (Post, Camera, Film, etc.)
- **HTTP handlers**: `/backend/server/` contains REST API implementation
- **Data layer**: `/backend/postgres/` for primary storage, `/backend/redis/` for caching
- **Vector operations**: `/backend/weaviate/` handles image similarity using embeddings
- **Configuration**: `/backend/config/` centralizes all app configuration

### Frontend Structure
- **Pages**: `/web/app/` contains Next.js route pages
- **Components**: `/web/components/` has reusable React components with co-located CSS modules
- **API Integration**: Uses generated TypeScript client from `/api/clients/typescript/`
- **State Management**: Custom hooks in `/web/hooks/` for data fetching
- **Styling**: CSS Modules pattern with Mantine component library

### Data Pipeline
- **Ingestion**: Python scrapers collect data from external sources
- **Processing**: Dagster orchestrates ETL workflows in `/scrape/`
- **Storage**: PostgreSQL for relational data, S3 for images, Weaviate for embeddings
- **Analytics**: Events flow through Kafka to ClickHouse via the consumer service

### API Design
- **REST endpoints**: Follow conventional patterns (/posts, /cameras, /films)
- **OpenAPI spec**: Auto-generated from Go struct annotations
- **Authentication**: Basic auth for admin endpoints
- **Pagination**: Cursor-based pagination for large result sets

## Development Workflow

1. **Backend changes**: Modify Go code, run tests with `make test`, update swagger with `make swagger`
2. **Frontend changes**: Work in `/web/`, use `npm run dev` for hot reload
3. **API changes**: Regenerate clients with `make gen-client-*` after OpenAPI spec updates
4. **Database changes**: Add migrations to `/backend/postgres/migrations/`
5. **Infrastructure**: Use docker-compose files for consistent development environments

## Testing

- **Backend**: `go test ./...` runs all unit tests
- **Frontend**: Uses Next.js built-in testing via `npm test`
- **Integration**: Docker compose in `/test/k6/` for load testing
- **Database**: Test containers used in Go tests for isolated database testing