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
GO_OUT_DIR := backend/internal/gen/proto

.PHONY: proto
proto: proto-go

.PHONY: proto-go
proto-go:
	@echo "Generating Go proto files..."
	protoc -I=$(PROTO_DIR) \
		--go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/analytics/v1/event.proto
