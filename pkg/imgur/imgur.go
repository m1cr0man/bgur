package imgur

import (
	"golang.org/x/oauth2"
)

type ImgurAPI struct {
	authConfig *oauth2.Config
}
