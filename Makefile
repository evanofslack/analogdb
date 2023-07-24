.PHONY: up upd down build log

backend :
	cd backend && make upd

web :
	cd web && make upd

scraper :
	cd scraper && make upd
