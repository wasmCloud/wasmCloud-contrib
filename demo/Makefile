setup:
	docker run --name wasmcloud-pg -d --rm -p 5432:5432 -e POSTGRES_PASSWORD=postgres postgres:16-alpine
	wash config put pg-task-db \
	POSTGRES_HOST=127.0.0.1 \
	POSTGRES_PORT=5432 \
	POSTGRES_USERNAME=postgres \
	POSTGRES_PASSWORD=postgres \
	POSTGRES_DATABASE=postgres \
	POSTGRES_TLS_REQUIRED=false
	while true; do nc -zv 127.0.0.1 5432 && break; echo 'waiting for port 5432'; sleep 1; done
	sleep 5
	cat components/task-manager/tasks.sql | PGPASSWORD=postgres psql -Upostgres -h 127.0.0.1

build:
	cd providers/http-router && wash build
	cd components/task-manager && wash build
	cd components/image-processor && wash build
	cd components/image-analyzer && wash build

deploy:
	wash app deploy ./wadm.yaml

sync:
	cd providers/http-router && wit-deps
	cd components/task-manager && wit-deps
	cd components/image-processor && wit-deps
	cd components/image-analyzer && wit-deps
	go generate ./...

clean:
	wash app delete ./wadm.yaml
	docker stop wasmcloud-pg
	wash config del pg-task-db
	wash drain all

.PHONY: setup build deploy sync clean
