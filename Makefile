.PHONY: up up-d down log run dev test psql

up :
	docker-compose up 

up-d :
	docker-compose up -d

down : 
	docker-compose down --remove-orphans

log :
	docker-compose logs --tail=0 --follow

run :
	cd cmd/analogdb && go run .

dev :
	docker-compose up -d && cd cmd/analogdb && go run .

test :
	docker-compose up -d && go test ./...

psql :
	docker exec -it  postgres psql -U postgres analog-local