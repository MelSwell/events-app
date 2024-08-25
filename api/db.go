package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v4/stdlib"
)

func (app *application) ConnectToDB() (*sql.DB, error) {
	db, err := openDB(app.DSN)
	if err != nil {
		return nil, err
	}

	log.Println("Database connection established")
	return db, nil
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
}

// func (app *application) runMigrations() error {
// 	driver, err := pgx.WithInstance(app.DB, &pgx.Config{})
// 	if err != nil {
// 		return fmt.Errorf("failed to create migration driver: %v", err)
// 	}

// 	m, err := migrate.NewWithDatabaseInstance("file://migrations", "db", driver)
// 	if err != nil {
// 		return fmt.Errorf("failed to create migration instance: %v", err)
// 	}

// 	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
// 		return fmt.Errorf("failed to run migrations: %v", err)
// 	}

// 	log.Println("Migrations complete")
// 	return nil
// }
