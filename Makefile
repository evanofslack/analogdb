.PHONY: backend
backend :
	cd backend && make upd

.PHONY: web
web :
	cd web && make upd

.PHONY: scraper
scraper :
	cd scraper && make upd

PROTO_DIR := proto
BACKEND_GO_OUT_DIR := backend/internal/gen/proto
CONSUMER_GO_OUT_DIR := consumer/internal/gen/proto

.PHONY: proto
proto: proto-go

.PHONY: proto-go
proto-go: proto-backend proto-consumer

.PHONY: proto-backend
proto-backend:
	@echo "Generating Go proto files for backend..."
	protoc -I=$(PROTO_DIR) \
		--go_out=$(BACKEND_GO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(BACKEND_GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/analytics/v1/event.proto

.PHONY: proto-consumer
proto-consumer:
	@echo "Generating Go proto files for consumer..."
	protoc -I=$(PROTO_DIR) \
		--go_out=$(CONSUMER_GO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(CONSUMER_GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/analytics/v1/event.proto

.PHONY: swagger
swagger:
	@echo "Generating Swagger documentation..."
	cd backend && swag init --dir ./ --generalInfo ./server/server.go --output ./docs
	@echo "Copying Swagger spec files to top-level api directory..."
	mkdir -p api
	cp backend/docs/swagger.json api/
	cp backend/docs/swagger.yaml api/
	@echo "Swagger documentation generated and copied to api/"

.PHONY: gen-client-python
gen-client-python:
	openapi-generator-cli generate -i api/swagger.yaml -g python -o api/clients/python \
		--additional-properties=packageName=analogdb_generated,projectName=analogdb-generated
	sed -i.bak 's/license = "NoLicense"/license = "MIT"/' api/clients/python/pyproject.toml
	rm -f api/clients/python/pyproject.toml.bak
	cd scrape/packages/analogdb && uv sync

