package imgur

import (
	oa2 "bgur/pkg/oauth2"
	"golang.org/x/oauth2"
)

type API struct {
	*oa2.API
}

func (i *API) Authorize(tokenFile string) error {
	i.SetConfig(&oauth2.Config{
		ClientID:     "825af7b91a9dfbf",
		ClientSecret: ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://api.imgur.com/oauth2/authorize",
			TokenURL:  "https://api.imgur.com/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	})
	return i.API.Authorize(tokenFile)
}

func NewAPI() *API {
	return &API{API: &oa2.API{}}
}
