
## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v
	go run github.com/swaggo/swag/cmd/swag@latest fmt


## build: build local the cmd/api application
.PHONY: build/local
build/local:
	go build -o=/tmp/bin/api-server ./cmd/api-server
	

## run: run local the cmd/api application
.PHONY: run/local
run/local: build/local
	/tmp/bin/api-server -cfg .local.env -prettyLog


## run/local/live: run the application with reloading on file changes
.PHONY: run/local/live
run/local/live:
	go run github.com/cosmtrek/air@v1.43.0 \
		--build.cmd "make build" --build.bin "/tmp/bin/api-server -cfg .local.env -prettyLog" --build.delay "100" \
		--build.exclude_dir "" \
		--build.include_ext "go, tpl, tmpl, html, css, scss, js, ts, sql, jpeg, jpg, gif, png, bmp, svg, webp, ico, json, yaml, yml" \
		--misc.clean_on_exit "true"


## run/stage: run all services in a docker containers(docker compose)
.PHONY: run/stage
run/stage:
	docker compose up -d --build


## stop/stage: stop all services in a docker containers(docker compose)
.PHONY: stop/stage
stop/stage:
	docker compose down


## run/stage/db: run db(postgres) in a docker container(docker compose)
.PHONY: run/stage/db
run/stage/db:
	docker compose up db -d


## run/stage/mock-people-service: run mock-people-service in a docker container(docker compose)
.PHONY: run/stage/mock-people-service
run/stage/mock-people-service:
	docker compose up mock_people_service -d


## run/local/mock-people-service: run local the script/mock-people-service application
.PHONY: run/local/mock-people-service
run/local/mock-people-service:
	cd ./scripts/mock-people-service && \
		PORT=8081 npm run start


## migrations/new name=$1: create a new database migration
.PHONY: migrations/new
migrations/new:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest create -seq -ext=.sql -dir=./assets/migrations ${name}


## migrations/up: apply all up database migrations
.PHONY: migrations/up
migrations/up:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path=./assets/migrations -database="postgres://${DB_DSN}" up


## migrations/down: apply all down database migrations
.PHONY: migrations/down
migrations/down:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path=./assets/migrations -database="postgres://${DB_DSN}" down


## migrations/goto version=$1: migrate to a specific version number
.PHONY: migrations/goto
migrations/goto:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path=./assets/migrations -database="postgres://${DB_DSN}" goto ${version}


## migrations/force version=$1: force database migration
.PHONY: migrations/force
migrations/force:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path=./assets/migrations -database="postgres://${DB_DSN}" force ${version}


## migrations/version: print the current in-use migration version
.PHONY: migrations/version
migrations/version:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest -path=./assets/migrations -database="postgres://${DB_DSN}" version


## gen/api: generate swagger api spec
.PHONY: gen/api
gen/api:
	go run github.com/swaggo/swag/cmd/swag@latest init -dir ./cmd/api-server --parseDependency --parseInternal

