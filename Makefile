.PHONY: up up-d down log run dev test psql

up :
	docker-compose -f docker-compose-dev.yml up

up-d :
	docker-compose -f docker-compose-dev.yml up -d

down :
	docker-compose down --remove-orphans

build :
	docker-compose -f docker-compose-dev.yml up -d --force-recreate --build

db :
	docker-compose -f docker-compose-dev.yml up -d postgres

log :
	docker-compose logs --tail=0 --follow

test :
	docker-compose -f docker-compose-dev.yml up -d && go test ./...

psql :
	docker exec -it postgres psql -U postgres analogdb
