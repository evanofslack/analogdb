mod backend
mod consumer
mod web
mod scrape
mod infra

proto_dir := "proto"

[private]
default:
    @just --list --list-submodules

# Create the shared docker network if it is missing
network:
    docker network inspect analogdb >/dev/null 2>&1 || docker network create analogdb

# Install deps for every service
setup: network
    cd backend && go mod download
    cd consumer && go mod download
    cd scrape && uv sync --dev
    cd web && npm ci

# Start the full stack in the background
up: network
    just backend up
    just consumer up
    just web up
    just infra up

# Stop the full stack
down:
    -just infra down
    -just web down
    -just consumer down
    -just backend down

# List analogdb containers
ps:
    docker ps --filter network=analogdb --format 'table {{{{.Names}}\t{{{{.Status}}\t{{{{.Ports}}'

# Run Go and Python tests
test:
    just backend test
    just consumer test
    just scrape test

# Lint every service
lint:
    just backend lint
    just consumer lint
    just scrape lint
    just web lint

# Format every service
fmt:
    just backend fmt
    just consumer fmt
    just scrape fmt
    just web fmt

# Lint, test and check the swagger spec
check: lint test swagger-check

# Generate Go code from protobuf for backend and consumer
proto: (_proto "backend/internal/gen/proto") (_proto "consumer/internal/gen/proto")

_proto out:
    protoc -I={{ proto_dir }} \
        --go_out={{ out }} --go_opt=paths=source_relative \
        --go-grpc_out={{ out }} --go-grpc_opt=paths=source_relative \
        {{ proto_dir }}/analytics/v1/event.proto

# Generate the OpenAPI spec into api/
swagger:
    cd backend && swag init --dir ./ --generalInfo ./server/server.go --output ./docs
    mkdir -p api
    cp backend/docs/swagger.json backend/docs/swagger.yaml api/

# Fail if the committed OpenAPI spec is stale
swagger-check: swagger
    git diff --exit-code api/swagger.json api/swagger.yaml

# Generate the Python and TypeScript API clients
gen-clients: gen-client-python gen-client-typescript

# Generate the Python API client
gen-client-python:
    openapi-generator-cli generate -i api/swagger.yaml -g python -o api/clients/python \
        --additional-properties=packageName=analogdb_generated,projectName=analogdb-generated
    sed -i.bak 's/license = "NoLicense"/license = "MIT"/' api/clients/python/pyproject.toml
    rm -f api/clients/python/pyproject.toml.bak
    cd scrape/packages/analogdb && uv sync

# Generate the TypeScript API client
gen-client-typescript:
    openapi-generator-cli generate -i api/swagger.yaml -g typescript-fetch -o api/clients/typescript \
        --additional-properties=npmName=analogdb-generated,npmVersion=1.0.0,withSeparateModelsAndApi=true,typescriptThreePlus=true
    cd api/clients/typescript && npm install
    cd web && npm install

# Run a k6 load test script
[working-directory('test/k6')]
load script:
    ./run.sh {{ script }}

# Remove build and test artifacts
clean:
    just backend clean
    just consumer clean
    just scrape clean
    just web clean

# Stop everything and delete volumes, the network and the bench database
[confirm("Delete all analogdb containers, volumes and the network?")]
nuke:
    -cd infra && docker compose down -v --remove-orphans
    -cd web && docker compose down -v --remove-orphans
    -cd consumer && docker compose down -v --remove-orphans
    -cd backend && docker compose down -v --remove-orphans
    -cd scrape && docker compose down -v --remove-orphans
    -docker rm -f analogdb-bench
    -docker network rm analogdb
