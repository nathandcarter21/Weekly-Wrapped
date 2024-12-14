package main

import (
	"database/sql"
	"log"

	"github.com/joho/godotenv"
	"github.com/robfig/cron"
)

var db *sql.DB

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("error loading .env file")
	}

	db = dbConnect()
	defer db.Close()

	c := cron.New()
	err = c.AddFunc("0 17 * * 5", loadData)
	if err != nil {
		log.Fatal(err)
	}

	c.Start()
	go Serve()

	select {}
}
