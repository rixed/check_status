# vim: ft=make
.PHONY: up down provision install

SOURCES = checker.go db.go main.go server.go

all: docker

docker: Dockerfile $(SOURCES) entrypoint.sh .dockerignore
	docker build --progress=plain -t check_status -f Dockerfile .
	touch $@

up:
	docker compose -p check_status up -d

down:
	docker compose -p check_status down

provision: content.sql
	docker exec -i check_status-db-1 psql -U postgres check_status < content.sql

install: nginx/check-status
	sudo cp $< /etc/nginx/sites-available
	sudo ln -sf /etc/nginx/sites-available/check-status /etc/nginx/sites-enabled/check-status
	sudo mkdir -p /var/log/nginx/check-status
