package imgur

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

func (i *ImgurAPI) WaitForAuth(stateStr string) (*oauth2.Token, error) {
	server := &http.Server{Addr: ":8099"}

	// Decode CallbackPage
	// It will never fail, it's hard coded
	CallbackPageParsed, _ := base64.StdEncoding.DecodeString(CallbackPage)

	var token *oauth2.Token

	http.HandleFunc("/oauthcallback", func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodGet {
			resp.Header().Set("Content-Type", "text/html; charset=utf-8")
			resp.Header().Set("Content-Encoding", "gzip")
			if _, err := resp.Write(CallbackPageParsed); err != nil {
				fmt.Println("Failed to write GET response: ", err)
			}

		} else if req.Method == http.MethodPost {

			if err := req.ParseForm(); err != nil {
				fmt.Println("Failed to parse POST form data: ", err)
				return
			}

			// Check state
			if req.FormValue("state") != stateStr {
				fmt.Println("STATE STRING DOES NOT MATCH! You have probably been hacked." +
					"Check your system security and try again")
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

			_ = server.Shutdown(context.Background())
		}
	})

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		return nil, err
	}

	return token, nil
}

func (i *ImgurAPI) Authorize() error {
	i.authConfig = &oauth2.Config{
		ClientID:     "825af7b91a9dfbf",
		ClientSecret: ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://api.imgur.com/oauth2/authorize",
			TokenURL:  "https://api.imgur.com/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	}

	randBytes := make([]byte, 24)
	if _, err := rand.Read(randBytes); err != nil {
		return err
	}
	stateStr := hex.EncodeToString(randBytes)

	url := i.authConfig.AuthCodeURL(stateStr, oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("response_type", "token"))
	if err := browser.OpenURL(url); err != nil {
		fmt.Println("Open this URL to authorise the app: ", url)
	}
	fmt.Println("Authorisation page opened. Check your browser.")

	token, err := i.WaitForAuth(stateStr)

	if err != nil {
		return fmt.Errorf("failed to read authorisation token: %s", err)
	}

	jsonData, _ := json.Marshal(token)

	if err = ioutil.WriteFile("token.json", jsonData, 0440); err != nil {
		return fmt.Errorf("failed to write token json file: %s", err)
	}

	return nil
}
