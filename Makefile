.PHONY: up down validate log

validate: 
	docker-compose config --quiet

up :
	docker-compose up -d

down : 
	docker-compose down

log :
	docker-compose logs --tail=0 --follow