package main

import (
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"

	"warehousecore/internal/migrations"

	_ "github.com/lib/pq"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL required")
	}
	dir := flag.String("dir", "migrations", "migrations directory")
	flag.Parse()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()
	if err := migrations.ApplyMigrations(db, *dir); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}
	// Apply seeds if present
	if err := migrations.ApplySeeds(db, filepath.Join(*dir, "seeds")); err != nil {
		log.Fatalf("apply seeds: %v", err)
	}
	log.Println("migrations complete")
}
