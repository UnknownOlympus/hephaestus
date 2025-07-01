package main

import (
	"log"

	"github.com/Houeta/us-api-provider/internal/config"
	"github.com/Houeta/us-api-provider/internal/repository"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose"
)

func main() {
	cfg := config.MustLoad()

	dbpool, dbErr := repository.NewDatabase(
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Dbname)
	if dbErr != nil {
		log.Fatalf("Failed to connect to DB: %v", dbErr)
	}

	dtb := stdlib.OpenDBFromPool(dbpool)
	if migrationErr := goose.Up(dtb, "migrations"); migrationErr != nil {
		log.Fatal(migrationErr)
	}
	defer dbpool.Close()

	log.Println("✅ Migrations applied successfully")
}
