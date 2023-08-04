.PHONY: up upd down build db infra mon log test

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
