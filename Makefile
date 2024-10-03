CC=go

lint:
	@golangci-lint run -c ./golangci.yml ./...

build: lint
	@$(CC) build -o bin/sso ./cmd/sso

run: build
	@./bin/sso --config=./config/local.yml

test:
	@$(CC) test -v ./tests/...

migrate_up:
	@$(CC) run ./cmd/migrator --storage-path=./storage/sso.db --migrations-path=./migrations/sqlite/

migrate_down:
	@$(CC) run ./cmd/migrator --migration-type=down --storage-path=./storage/sso.db --migrations-path=./migrations/sqlite/

migrate_up_postgres:
	@$(CC) run ./cmd/migrator --db=postgres --migrations-path=./migrations/postgres/

migrate_down_postgres:
	@$(CC) run ./cmd/migrator --migration-type=down --db=postgres --migrations-path=./migrations/postgres/ --storage-path=db:5432

migrate_test_down:
	@$(CC) run ./cmd/migrator --migration-type=down --storage-path=./storage/sso.db --migrations-path=./tests/migrations --migrations-table=migrations_test