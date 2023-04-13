.PHONY: up up-d down log run dev test psql

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

prod :
	docker-compose -f docker-compose.yml up -d

log :
	docker-compose -f docker-compose-dev.yml logs --follow

test :
	docker-compose -f docker-compose-dev.yml up -d && go test ./...

psql :
	docker exec -it postgres psql -U postgres analogdb
