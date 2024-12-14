package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/securecookie"
)

func Serve() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/code", codeHandler)
	http.HandleFunc("/signup", signUpHandler)

	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := readCookie(w, r)
	if err != nil {
		authRedirect(w, r)
		return
	}

	userID, err := getUser(accessToken.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	tmplPath := ""
	dbUser, err := getDBUser(db, userID)

	if userID == "" || dbUser == "" || err != nil {
		tmplPath = filepath.Join("templates/signup.html")
	} else {
		tmplPath = filepath.Join("templates/home.html")
	}

	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		http.Error(w, "Error parsing template", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "Error executing template", http.StatusInternalServerError)
	}
}

func codeHandler(w http.ResponseWriter, r *http.Request) {
	u := r.URL
	params, err := url.ParseQuery(u.RawQuery)

	if err != nil {
		log.Fatal(err)
	}

	code := params.Get("code")
	qerr := params.Get("error")

	if qerr != "" {
		log.Fatal(qerr)
	}

	if code == "" {
		log.Fatal("err: no code")
	}

	accessToken, err := requestAccessToken(code)
	if err != nil || accessToken == nil {
		log.Fatal(err)
	}

	setCookie(w, *accessToken)

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func signUpHandler(w http.ResponseWriter, r *http.Request) {
	accessToken, err := readCookie(w, r)
	if err != nil {
		log.Fatal(err)
	}

	id, err := getUser(accessToken.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	err = signUp(db, id, accessToken.RefreshToken)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(w, `<p>Stay Tuned for Updates</p>`)
}

func authRedirect(w http.ResponseWriter, r *http.Request) {
	u := url.URL{
		Scheme: "https",
		Host:   "accounts.spotify.com",
		Path:   "/authorize",
	}

	query := u.Query()
	query.Set("client_id", os.Getenv("SPOTIFY_CLIENT"))
	query.Set("response_type", "code")
	query.Set("redirect_uri", "http://localhost:3000/code")
	query.Set("scope", "user-top-read")
	u.RawQuery = query.Encode()

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

func readCookie(w http.ResponseWriter, r *http.Request) (*AccessToken, error) {
	cookie, err := r.Cookie("accessToken")
	if err != nil {
		return nil, errors.New("no access token")
	}
	var value string

	var s = securecookie.New([]byte(os.Getenv("COOKIE_SALT")), []byte(os.Getenv("COOKIE_HASH")))
	err = s.Decode("accessToken", cookie.Value, &value)
	if err != nil {
		return nil, err
	}

	var accessToken *AccessToken
	err = json.Unmarshal([]byte(value), &accessToken)
	if err != nil {
		return nil, err
	}

	if accessToken.ExpiresAt.Before(time.Now()) {
		accessToken, err = refreshAccessToken(*accessToken)
		if err != nil {
			log.Fatal(err)
		}

		setCookie(w, *accessToken)
	}

	return accessToken, nil
}

func setCookie(w http.ResponseWriter, accessToken AccessToken) error {
	var cookieSalt = []byte(os.Getenv("COOKIE_SALT"))
	var cookieHash = []byte(os.Getenv("COOKIE_HASH"))
	var s = securecookie.New(cookieSalt, cookieHash)

	jsonToken, err := json.Marshal(accessToken)
	if err != nil {
		return err
	}

	encoded, err := s.Encode("accessToken", string(jsonToken))
	if err != nil {
		return err
	}

	cookie := &http.Cookie{
		Name:     "accessToken",
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Now().Add(72 * time.Hour),
	}
	http.SetCookie(w, cookie)

	return nil
}

func requestAccessToken(code string) (*AccessToken, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "accounts.spotify.com",
		Path:   "/api/token",
	}

	bytesAuth := []byte(os.Getenv("SPOTIFY_CLIENT") + ":" + os.Getenv("SPOTIFY_SECRET"))
	encodedAuth := base64.StdEncoding.EncodeToString(bytesAuth)
	headers := map[string]string{
		"Authorization": "Basic " + encodedAuth,
		"Content-Type":  "application/x-www-form-urlencoded",
	}

	data := url.Values{
		"grant_type":   {"authorization_code"},
		"code":         {code},
		"redirect_uri": {"http://localhost:3000/code"},
	}

	body := bytes.NewBufferString(data.Encode())

	res, err := makeApiCall("POST", u, headers, body)
	if err != nil {
		return nil, err
	}

	var accessToken AccessToken
	err = json.Unmarshal(res, &accessToken)
	if err != nil {
		return nil, err
	}

	if accessToken.Error != "" {
		return nil, errors.New(accessToken.Error)
	}

	accessToken.ExpiresAt = time.Now().Add(time.Duration(accessToken.ExpiresIn-100) * time.Second)

	return &accessToken, nil
}

func refreshAccessToken(prevToken AccessToken) (*AccessToken, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "accounts.spotify.com",
		Path:   "/api/token",
	}

	bytesAuth := []byte(os.Getenv("SPOTIFY_CLIENT") + ":" + os.Getenv("SPOTIFY_SECRET"))
	encodedAuth := base64.StdEncoding.EncodeToString(bytesAuth)
	headers := map[string]string{
		"Authorization": "Basic " + encodedAuth,
		"Content-Type":  "application/x-www-form-urlencoded",
	}

	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {prevToken.RefreshToken},
	}

	body := bytes.NewBufferString(data.Encode())

	res, err := makeApiCall("POST", u, headers, body)
	if err != nil {
		return nil, err
	}

	var newToken AccessToken
	err = json.Unmarshal(res, &newToken)
	if err != nil {
		return nil, err
	}

	newToken.ExpiresAt = time.Now().Add(time.Duration(newToken.ExpiresIn-100) * time.Second)
	newToken.RefreshToken = prevToken.RefreshToken

	return &newToken, nil
}

func getUser(accessToken string) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.spotify.com",
		Path:   "/v1/me",
	}

	headers := map[string]string{
		"Authorization": "Bearer " + accessToken,
	}

	res, err := makeApiCall("GET", u, headers, nil)
	if err != nil {
		return "", err
	}

	var spotifyId *SpotifyID
	err = json.Unmarshal(res, &spotifyId)
	if err != nil {
		return "", err
	}

	return spotifyId.ID, nil
}

func makeApiCall(method string, u url.URL, headers map[string]string, reqBody io.Reader) ([]byte, error) {
	req, err := http.NewRequest(method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	switch res.StatusCode {
	case http.StatusOK:
		return resBody, nil
	default:
		return nil, errors.New(string(resBody))
	}
}
