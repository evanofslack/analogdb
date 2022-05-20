.PHONY: validate up up-d down log dev test psql

validate: 
	docker-compose config --quiet

up :
	docker-compose up 

up-d :
	docker-compose up -d

down : 
	docker-compose down --remove-orphans

log :
	docker-compose logs --tail=0 --follow

dev :
	docker-compose up -d && cd server && go run .

test :
	docker-compose up -d && cd server && go test ./...

psql :
	docker exec -it  postgres psql -U postgres analog-local