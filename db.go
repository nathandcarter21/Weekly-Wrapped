package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func dbConnect() *sql.DB {
	dsn := "root:password@tcp(127.0.0.1:3306)/weekly-wrapped"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("error opening database:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("error pinging database:", err)
	}

	return db
}

func signUp(db *sql.DB, spotifyId, refreshToken string) error {
	stmt, err := db.Prepare("insert into users (spotify_id, refresh_token) values (?, ?)")
	if err != nil {
		return fmt.Errorf("error preparing statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(spotifyId, refreshToken)
	if err != nil {
		return fmt.Errorf("error executing statement: %v", err)
	}

	return nil
}

func getDBUser(db *sql.DB, spotifyId string) (string, error) {
	query := `SELECT spotify_id FROM users WHERE spotify_id = ?`

	var userID string
	err := db.QueryRow(query, spotifyId).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user with ID %s not found", userID)
		}
		return "", fmt.Errorf("error querying user: %w", err)
	}

	return userID, nil
}
