.PHONY: up upd down build db infra mon log test

up :
	docker compose up

upd :
	docker compose up -d

down :
	docker compose down --remove-orphans

build :
	docker compose up -d --build

log :
	docker compose logs --follow
