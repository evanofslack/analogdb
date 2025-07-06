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
