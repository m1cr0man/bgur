package oauth2

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pkg/browser"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type API struct {
	authConfig *oauth2.Config
	AuthUrl    string
	Username   string
	Client     *http.Client
}

type TokenWithUsername struct {
	Token    *oauth2.Token `json:"token"`
	Username string        `json:"username,omitempty"`
}

func (i *API) AuthFromWeb() (*oauth2.Token, error) {

	// Generate a random string for the state value
	// Prevents XSS
	randBytes := make([]byte, 24)
	if _, err := rand.Read(randBytes); err != nil {
		return nil, err
	}
	stateStr := hex.EncodeToString(randBytes)

	// Open the provider auth page
	url := i.authConfig.AuthCodeURL(stateStr, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("response_type", "token"))
	if err := browser.OpenURL(url); err != nil {
		fmt.Println("Open this URL to authorise the app: ", url)
	} else {
		fmt.Println("Authorisation page opened. Check your browser.")
	}

	// Decode CallbackPage
	// It will never fail, it's hard coded
	CallbackPageParsed, _ := base64.StdEncoding.DecodeString(CallbackPage)
	tokenChannel := make(chan oauth2.Token)

	http.HandleFunc(i.AuthUrl, func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			resp.Header().Set("Content-Type", "text/html;charset=UTF-8")
			resp.Header().Set("Content-Encoding", "gzip")

			// Check state
			if req.URL.Query().Get("state") != stateStr {
				fmt.Println("STATE STRING DOES NOT MATCH! You have probably been hacked. " +
					"Check your system security and try again")
				resp.WriteHeader(401)
				close(tokenChannel)
				return
			}

			if _, err := resp.Write(CallbackPageParsed); err != nil {
				fmt.Println("Failed to write GET response: ", err)
				close(tokenChannel)
			}

		} else if req.Method == http.MethodPost {

			if err := req.ParseForm(); err != nil {
				fmt.Println("Failed to parse POST form data: ", err)
				close(tokenChannel)
				return
			}

			// Parse form data to an Oauth2 token
			expiry, _ := strconv.Atoi(req.FormValue("expires_in"))
			tokenChannel <- oauth2.Token{
				Expiry:       time.Now().Add(time.Second * time.Duration(expiry)),
				TokenType:    req.FormValue("token_type"),
				AccessToken:  req.FormValue("access_token"),
				RefreshToken: req.FormValue("refresh_token"),
			}

			close(tokenChannel)
			resp.WriteHeader(201)
		}
	})

	token, ok := <-tokenChannel
	if !ok {
		return nil, fmt.Errorf("failed to get token")
	}

	return &token, nil
}

func (i *API) AuthFromFile(tokenFile string) (token TokenWithUsername, err error) {

	// Try reading the token from the file
	tokenData, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(tokenData, &token)
	return
}

func (i *API) SaveAuth(tokenFile string, token TokenWithUsername) error {
	jsonData, _ := json.Marshal(token)
	return ioutil.WriteFile(tokenFile, jsonData, 0640)
}

func (i *API) SetConfig(config *oauth2.Config) {
	i.authConfig = config
}

func (i *API) Authorise(tokenFile string) error {
	token, err := i.AuthFromFile(tokenFile)

	// Don't complain, just do web auth
	if err != nil || token.Token == nil || token.Token.AccessToken == "" {
		err = nil
		newToken, err := i.AuthFromWeb()

		if err != nil {
			return fmt.Errorf("failed to get authorisation token from Imgur: %s", err)
		}
		token = TokenWithUsername{
			newToken,
			"",
		}
	}

	// Force a token refresh so that we can get the username
	if token.Username == "" {
		token.Token.Expiry = time.Now().Add(-time.Hour)
	}
	source := i.authConfig.TokenSource(context.Background(), token.Token)
	newToken, err := source.Token()
	if err != nil {
		return err
	}
	token.Token = newToken

	// Set up authenticated http client
	i.Client = oauth2.NewClient(context.Background(), source)

	// If token was refreshed set the username
	if newUsername, ok := newToken.Extra("account_username").(string); ok {
		token.Username = newUsername
	}
	i.Username = token.Username

	// Save the token
	if err = i.SaveAuth(tokenFile, token); err != nil {
		return fmt.Errorf("failed to save auth data: %s", err)
	}

	fmt.Println("Token received!")

	return nil
}

func NewAPI(authUrl string) *API {
	return &API{
		AuthUrl: authUrl,
	}
}
