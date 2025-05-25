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
KOTLIN_OUT_DIR := analytics/src/main/java

.PHONY: proto
proto: proto-go proto-cp-kotlin

.PHONY: proto-go
proto-go:
	@echo "Generating Go proto files..."
	protoc -I=$(PROTO_DIR) \
		--go_out=$(GO_OUT_DIR) --go_opt=paths=source_relative \
		--go-grpc_out=$(GO_OUT_DIR) --go-grpc_opt=paths=source_relative \
		$(PROTO_DIR)/analytics/v1/event.proto

.PHONY: proto-kotlin
proto-kotlin:
	@echo "Generating Kotlin proto files..."
	mkdir -p $(KOTLIN_OUT_DIR)
	protoc -I=$(PROTO_DIR) \
		--java_out=$(KOTLIN_OUT_DIR) \
		$(PROTO_DIR)/analytics/v1/event.proto

.PHONY: proto-cp-kotlin
proto-cp-kotlin:
	@echo "Generating Kotlin proto files..."
	cp -r $(PROTO_DIR) analytics/proto/
