package main

import (
	"encoding/json"
	"log"
	"net/url"
)

func loadData() {
	users, err := getUsers(db)
	if err != nil {
		log.Fatal(err)
	}

	for id, token := range users {
		accessToken, err := refreshAccessToken(token)
		if err != nil {
			log.Fatal(err)
		}

		vars := map[string]string{}
		vars["time_range"] = "long_term"
		vars["seed_type"] = "artists"

		artists, err := getUsersTopItems(accessToken.AccessToken, vars)
		if err != nil {
			log.Fatal(err)
		}

		vars["seed_type"] = "tracks"
		songs, err := getUsersTopItems(accessToken.AccessToken, vars)
		if err != nil {
			log.Fatal(err)
		}

		_ = addWrapped(db, id, artists, songs)
	}
}

func getUsersTopItems(accessToken string, vars map[string]string) ([]string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.spotify.com",
		Path:   "/v1/me/top/" + vars["seed_type"],
	}

	query := u.Query()
	query.Set("time_range", vars["time_range"])
	query.Set("limit", "5")
	u.RawQuery = query.Encode()

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	res, err := makeApiCall("GET", u, headers, nil)
	if err != nil {
		return nil, err
	}

	var item TopItems
	err = json.Unmarshal(res, &item)
	if err != nil {
		return nil, err
	}

	items := []string{}

	for _, val := range item.Items {
		items = append(items, val.Name)
	}

	return items, nil
}
