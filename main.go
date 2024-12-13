package main

import (
	"database/sql"
	"log"

	"github.com/joho/godotenv"
)

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}

	db = dbConnect()
	defer db.Close()

	Serve()
}
