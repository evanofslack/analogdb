.PHONY: dev up upd down build log format

dev:
	npm run dev

up :
	docker-compose up

upd :
	docker-compose up -d

down :
	docker-compose down --remove-orphans

build :
	docker-compose up -d --force-recreate --build

log:
	docker-compose logs --follow

format:
	npx prettier --write .

