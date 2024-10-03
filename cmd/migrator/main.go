package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

const (
	migrationUp   = "up"
	migrationDown = "down"
)

func mustMigrateUp(m *migrate.Migrate) {
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")
			return
		}

		panic(err)
	}

	fmt.Println("migrations applied successfully")
}

func mustMigrateDown(m *migrate.Migrate) {
	if err := m.Down(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			fmt.Println("no migrations to apply")
			return
		}

		panic(err)
	}

	fmt.Println("migrations downed successfully")
}

// TODO: standalone docker migrator
func main() {
	var storagePath, migrationsPath, migrationsTable, migrationType, db string
	flag.StringVar(&migrationType, "migration-type", migrationUp, "migration type")
	flag.StringVar(&db, "db", "sqlite3", "database")
	flag.StringVar(&storagePath, "storage-path", "", "path to the storage")
	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&migrationsTable, "migrations-table", "migrations", "name of migrations table")
	flag.Parse()

	if storagePath == "" && db == "sqlite3" {
		panic("storage-path is required")
	}

	if migrationsPath == "" {
		panic("migrations-path is required")
	}

	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	m, err := migrate.New(
		sourceURL,
		dbUrl(db, storagePath, migrationsTable),
	)

	if err != nil {
		panic(err)
	}

	if migrationType == migrationDown {
		mustMigrateDown(m)
		return
	}

	mustMigrateUp(m)
}

func dbUrl(db, storagePath, migrationsTable string) string {
	switch db {
	case "sqlite3":
		return fmt.Sprintf("sqlite3://%s?x-migrations-table=%s", storagePath, migrationsTable)
	case "postgres":
		return fmt.Sprintf("postgres://postgres:password@%s/sso?sslmode=disable", storagePath)
	}

	return ""
}
