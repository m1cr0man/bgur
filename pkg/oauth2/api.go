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
	Client     *http.Client
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

	// Start a HTTP server to handle the oauth response from the client browser
	server := &http.Server{Addr: ":8099"}

	// Decode CallbackPage
	// It will never fail, it's hard coded
	CallbackPageParsed, _ := base64.StdEncoding.DecodeString(CallbackPage)

	var token *oauth2.Token

	http.HandleFunc("/oauthcallback", func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			resp.Header().Set("Content-Type", "text/html;charset=UTF-8")
			resp.Header().Set("Content-Encoding", "gzip")

			// Check state
			if req.URL.Query().Get("state") != stateStr {
				fmt.Println("STATE STRING DOES NOT MATCH! You have probably been hacked." +
					"Check your system security and try again")

				// Shutdown soon
				go func() {
					time.Sleep(time.Second)
					_ = server.Shutdown(context.Background())
				}()

				return
			}

			if _, err := resp.Write(CallbackPageParsed); err != nil {
				fmt.Println("Failed to write GET response: ", err)
			}

		} else if req.Method == http.MethodPost {

			if err := req.ParseForm(); err != nil {
				fmt.Println("Failed to parse POST form data: ", err)
				return
			}

			// Parse form data to an Oauth2 token
			expiry, _ := strconv.Atoi(req.FormValue("expires_in"))
			token = &oauth2.Token{
				Expiry:       time.Now().Add(time.Second * time.Duration(expiry)),
				TokenType:    req.FormValue("token_type"),
				AccessToken:  req.FormValue("access_token"),
				RefreshToken: req.FormValue("refresh_token"),
			}

			resp.WriteHeader(201)

			// Shutdown soon
			go func() {
				time.Sleep(time.Second)
				_ = server.Shutdown(context.Background())
			}()
		}
	})

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return nil, err
	}

	return token, nil
}

func (i *API) AuthFromFile(tokenFile string) (token *oauth2.Token, err error) {
	token = &oauth2.Token{}

	// Try reading the token from the file
	tokenData, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return
	}
	err = json.Unmarshal(tokenData, token)
	return
}

func (i *API) SaveAuth(tokenFile string, token *oauth2.Token) error {
	jsonData, _ := json.Marshal(token)
	return ioutil.WriteFile(tokenFile, jsonData, 0640)
}

func (i *API) SetConfig(config *oauth2.Config) {
	i.authConfig = config
}

func (i *API) Authorize(tokenFile string) error {
	var token *oauth2.Token

	token, err := i.AuthFromFile(tokenFile)

	// Don't complain, just do web auth
	if err != nil {
		err = nil
		token, err = i.AuthFromWeb()
	}

	if err != nil {
		return fmt.Errorf("failed to read authorisation token: %s", err)
	}

	// Set up authenticated http client
	i.Client = i.authConfig.Client(context.Background(), token)

	// Save the token
	if err = i.SaveAuth(tokenFile, token); err != nil {
		return fmt.Errorf("failed to save auth data: %s", err)
	}

	fmt.Println("Token received!")

	return nil
}
