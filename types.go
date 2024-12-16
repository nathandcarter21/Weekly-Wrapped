package main

import "time"

type AccessToken struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"`
	ExpiresAt    time.Time `json:"expires_at"`
	SpotifyID    string    `json:"spotify_id"`
}

type SpotifyID struct {
	ID string `json:"id"`
}

type TopItems struct {
	Items []Item `json:"items"`
}

type Item struct {
	Name string `json:"name"`
}
