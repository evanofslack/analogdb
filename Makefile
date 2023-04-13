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

prod :
	docker-compose -f docker-compose.yml up -d

log :
	docker-compose -f docker-compose-dev.yml logs --tail=0 --follow

test :
	docker-compose -f docker-compose-dev.yml up -d && go test ./...

psql :
	docker exec -it postgres psql -U postgres analogdb
