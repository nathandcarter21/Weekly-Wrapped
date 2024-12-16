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
	dsn := "root:password@tcp(127.0.0.1:3306)/weekly-wrapped?parseTime=true"

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

	eSpotifyId, err := encrypt(spotifyId)
	if err != nil {
		return err
	}
	eRefreshT, err := encrypt(refreshToken)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(eSpotifyId, eRefreshT)
	if err != nil {
		return fmt.Errorf("error executing statement: %v", err)
	}

	return nil
}

func getDBUser(db *sql.DB, spotifyId string) (string, error) {
	query := `select spotify_id from users where spotify_id = ?`

	eUserId, err := encrypt(spotifyId)
	if err != nil {
		return "", err
	}

	var userID string
	err = db.QueryRow(query, eUserId).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("user with ID %s not found", eUserId)
		}
		return "", fmt.Errorf("error querying user: %w", err)
	}

	cUserId, err := decrypt(userID)
	if err != nil {
		return "", err
	}

	return cUserId, nil
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
		cId, err := decrypt(id)
		if err != nil {
			return nil, err
		}
		cToken, err := decrypt(token)
		if err != nil {
			return nil, err
		}
		users[cId] = cToken
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return users, nil
}

func addWrapped(db *sql.DB, spotifyID string, artists, songs []string) error {
	cSpotifyId, err := encrypt(spotifyID)
	if err != nil {
		return err
	}

	artistsJson, _ := json.Marshal(artists)
	songsJson, _ := json.Marshal(songs)

	query := `insert into wraps (spotify_id, date, songs, artists) values (?, ?, ?, ?)`
	_, err = db.Exec(query, cSpotifyId, time.Now().Format("2006-01-02"), songsJson, artistsJson)
	if err != nil {
		return err
	}
	return nil
}

func getWrappedDates(db *sql.DB, spotifyID string) ([]time.Time, error) {
	query := `select date from wraps where spotify_id = ? order by date desc`

	cSpotifyId, err := encrypt(spotifyID)
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(query, cSpotifyId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := []time.Time{}
	for rows.Next() {
		var date time.Time
		err := rows.Scan(&date)
		if err != nil {
			return nil, err
		}
		res = append(res, date)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return res, nil
}

func getWrapped(db *sql.DB, spotifyId string, date string) (artists, songs []string, err error) {
	query := `select artists, songs from wraps where spotify_id = ? and date = ?`

	cSpotifyId, err := encrypt(spotifyId)
	if err != nil {
		return nil, nil, err
	}

	var artistJson string
	var songJson string
	err = db.QueryRow(query, cSpotifyId, date).Scan(&artistJson, &songJson)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, err
		}
		return nil, nil, err
	}

	json.Unmarshal([]byte(artistJson), &artists)
	json.Unmarshal([]byte(songJson), &songs)

	return
}
