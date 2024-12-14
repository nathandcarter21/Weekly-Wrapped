package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

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
	query := `select spotify_id from users where spotify_id = ?`

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

func getUsers(db *sql.DB) (map[string]string, error) {
	query := `select spotify_id, refresh_token from users`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := map[string]string{}
	for rows.Next() {
		var id, token string
		err := rows.Scan(&id, &token)
		if err != nil {
			return nil, err
		}
		users[id] = token
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return users, nil
}

func addWrapped(db *sql.DB, spotifyID string, artists, songs []string) error {

	artistsJson, _ := json.Marshal(artists)
	songsJson, _ := json.Marshal(songs)

	query := `insert into wraps (spotify_id, date, songs, artists) values (?, ?, ?, ?)`
	_, err := db.Exec(query, spotifyID, time.Now().Format("2006-01-02"), songsJson, artistsJson)
	if err != nil {
		return err
	}
	return nil
}
