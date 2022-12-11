include .envrc

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'


.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N]' && read ans && [ $${ans:-N} = y ]


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go test -race -vet=off ./...
	go mod verify


# ==================================================================================== #
# BUILD
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	@go run ./cmd/api -db-dsn=${QUOTES_DB_DSN} -base-url=${BASE_URL} -jwt-secret-key=${JWT_SECRETE_KEY} -smtp-host=${SMTP_HOST} -smtp-port=${SMTP_PORT} -smtp-username=${SMTP_USERNAME} -smtp-password=${SMTP_PASSWORD} -smtp-from=${SMTP_FROM} -telegram-bot-token=${TELEGRAM_BOT_TOKEN} -telegram-channel-id=${TELEGRAM_CHANNEL_ID} -use-telegram=true -sudoers=${SUDOERS}

## build: build the cmd/api application
.PHONY: build
build:
	go mod verify
	go build -ldflags='-s' -o=./bin/api ./cmd/api
	GOOS=linux GOARCH=amd64 go build -ldflags='-s' -o=./bin/linux_amd64/api ./cmd/api

## run: run the api application
.PHONY: run
run: tidy build
	./bin/api -db-dsn=${QUOTES_DB_DSN} -base-url=${BASE_URL} -jwt-secret-key=${JWT_SECRETE_KEY}


# ==================================================================================== #
# SQL MIGRATIONS
# ==================================================================================== #


## db/psql: connect to the database using psql
.PHONY: db/psql
db/psql:
	psql ${QUOTES_DB_DSN}

## migrations/new name=$1: create a new database migration
.PHONY: migrations/new
migrations/new:
	go run -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest create -seq -ext=.sql -dir=./assets/migrations ${name}

## migrations/up: apply all database migrations
.PHONY: migrations/up
migrations/up: confirm
	@echo "Running up migrations..."
	migrate -path ./assets/migrations -database ${QUOTES_DB_DSN} up

## migrations/down: apply all down database migrations
.PHONY: migrations/down
migrations/down:
	migrate -path ./assets/migrations -database ${QUOTES_DB_DSN} down

## migrations/goto version=$1: migrate to a specific version number
.PHONY: migrations/goto
migrations/goto:
	migrate -path ./assets/migrations -database ${QUOTES_DB_DSN} goto ${version}

## migrations/force version=$1: force database migration
.PHONY: migrations/force
migrations/force:
	migrate -path ./assets/migrations -database ${QUOTES_DB_DSN} force ${version}

## migrations/version: print the current in-use migration version
.PHONY: migrations/version
migrations/version:
	migrate -path ./assets/migrations -database ${QUOTES_DB_DSN} version

production_host_ip = "52.66.239.39"

.PHONY: production/connect
production/connect:
	ssh quotes@${production_host_ip}

.PHONY: production/deploy/api
production/deploy/api:
	rsync -rP --delete ./bin/linux_amd64/api ./assets/migrations quotes@${production_host_ip}:~
	ssh -t quotes@${production_host_ip} 'migrate -path ~/migrations -database $$QUOTES_DB_DSN up'

.PHONY: production/configure/api.service
production/configure/api.service:
	rsync -P ./remote/production/api.service quotes@${production_host_ip}:~
	ssh -t quotes@${production_host_ip} '\
	sudo mv ~/api.service /etc/systemd/system/ \
	&& sudo systemctl enable api \
	&& sudo systemctl restart api \
	'

.PHONY: production/configure/caddyfile
production/configure/caddyfile:
	rsync -P ./remote/production/Caddyfile quotes@${production_host_ip}:~
	ssh -t quotes@${production_host_ip} '\
	sudo mv ~/Caddyfile /etc/caddy/ \
	&& sudo systemctl reload caddy \
	'


.PHONY: production/refresh/api
production/refresh/api: build production/deploy/api production/configure/api.service

