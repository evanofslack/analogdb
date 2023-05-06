.PHONY: up upd down build db infra log test prod

up :
	docker-compose -f docker-compose-dev.yml up

upd :
	docker-compose -f docker-compose-dev.yml up -d

down :
	docker-compose -f docker-compose-dev.yml down --remove-orphans

build :
	docker-compose -f docker-compose-dev.yml up -d --force-recreate --build

db :
	docker-compose -f docker-compose-dev.yml up -d postgres

infra :
	docker-compose -f docker-compose-dev.yml up -d postgres weaviate i2v-neural

log :
	docker-compose -f docker-compose-dev.yml logs --follow

test :
	go test ./...

prod :
	docker-compose -f docker-compose.yml up -d

