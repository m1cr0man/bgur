package imgur

import (
	"golang.org/x/oauth2"
	"net/http"
)

type ImgurAPI struct {
	authConfig *oauth2.Config
	client     *http.Client
}
