package main

import "time"

type AccessToken struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	TokenType        string    `json:"token_type"`
	Scope            string    `json:"scope"`
	ExpiresIn        int       `json:"expires_in"`
	ExpiresAt        time.Time `json:"expires_at"`
	Error            string    `json:"error"`
	ErrorDescription string    `json:"error_description"`
}

type SpotifyID struct {
	ID string `json:"id"`
}
