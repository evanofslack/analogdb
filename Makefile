.PHONY: up up-d down log dev test psql

up :
	docker-compose up 

up-d :
	docker-compose up -d

down : 
	docker-compose down --remove-orphans

log :
	docker-compose logs --tail=0 --follow

run :
	cd server/cmd/analogdb && go run .

dev :
	docker-compose up -d && cd server/cmd/analogdb && go run .

test :
	docker-compose up -d && cd server && go test ./...

psql :
	docker exec -it  postgres psql -U postgres analog-local