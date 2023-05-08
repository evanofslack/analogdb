.PHONY: up upd down build db infra mon log test web prod

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

mon :
	docker-compose -f docker-compose-dev.yml up -d prometheus grafana

log :
	docker-compose -f docker-compose-dev.yml logs --follow

test :
	go test ./...

web :
	cd web && npm run dev

prod :
	docker-compose -f docker-compose.yml up -d

