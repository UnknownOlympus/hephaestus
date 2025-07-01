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

	dbpool, err := repository.NewDatabase(
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.Dbname)
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer dbpool.Close()

	dtb := stdlib.OpenDBFromPool(dbpool)
	if err := goose.Up(dtb, "migrations"); err != nil {
		log.Fatal(err)
	}

	log.Println("✅ Migrations applied successfully")
}
