package main

import (
	"events-app/data/repository"
	"log"
)

type application struct {
	DSN  string
	Repo repository.DBRepo
}

func main() {
	var app = &application{}
	app.DSN = "postgres://user:password@localhost:5432/db"

	db, err := app.ConnectToDB()
	if err != nil {
		log.Fatalf("Failed to connect to db: %v", err)
	}
	defer db.Close()

	app.Repo = &repository.SqlRepo{DB: db}

	if err = app.Repo.RunMigrations("db"); err != nil {
		log.Fatal(err.Error())
	}

}
