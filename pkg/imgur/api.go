package imgur

import (
	"encoding/json"
	"fmt"
	oa2 "github.com/m1cr0man/bgur/pkg/oauth2"
	"golang.org/x/oauth2"
	"io/ioutil"
	"net/http"
)

type API struct {
	*oa2.API
	unauthedClient *http.Client
}

func getProcessor(res *http.Response, olderr error) (body []byte, err error) {
	if olderr != nil {
		return
	}

	if res.StatusCode > 299 {
		err = fmt.Errorf("failed to get %s. Status code %d", res.Request.URL, res.StatusCode)
		return
	}

	return ioutil.ReadAll(res.Body)
}

func (i *API) get(url string) (body []byte, err error) {
	return getProcessor(i.API.Client.Get(url))
}

func (i *API) getUnauthed(url string) (body []byte, err error) {
	return getProcessor(i.unauthedClient.Get(url))
}

func (i *API) Authorise(tokenFile string) error {
	i.SetConfig(&oauth2.Config{
		ClientID:     "825af7b91a9dfbf",
		ClientSecret: ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://api.imgur.com/oauth2/authorize",
			TokenURL:  "https://api.imgur.com/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	})
	return i.API.Authorise(tokenFile)
}

func (i *API) GetFolders(folderOwner string) (folders []Folder, err error) {
	body, err := i.get(fmt.Sprintf("https://api.imgur.com/3/account/%s/folders", folderOwner))
	if err != nil {
		return
	}

	var response FoldersResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}

	folders = response.Data
	return
}

func (i *API) GetAlbumImages(albumId string) (images []Image, err error) {
	body, err := i.get(fmt.Sprintf("https://api.imgur.com/3/album/%s/images", albumId))
	if err != nil {
		return
	}

	var response AlbumContentResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}

	images = response.Data
	return
}

func (i *API) GetFolderImages(folderOwner string, folderId int) (images []Image, err error) {
	body, err := i.get(fmt.Sprintf("https://api.imgur.com/3/account/%s/folders/%d/favorites",
		folderOwner, folderId))
	if err != nil {
		return
	}

	var response FolderContentResponse
	if err = json.Unmarshal(body, &response); err != nil {
		return
	}

	for _, item := range response.Data {

		// Skip ads
		if item.IsAd {
			continue

			// Folders can contain albums. Flatten out the albums
		} else if item.IsAlbum && item.Album != nil {

			// Albums are partially loaded already. For single image albums (aka anything
			// from the gallery) there is no need for extra requests
			if len(item.Images) == item.ImagesCount {
				images = append(images, item.Images...)

				// For real albums, load all the images
			} else {
				extraImages, err2 := i.GetAlbumImages(item.Id)
				if err2 != nil {
					err = err2
					return
				}
				images = append(images, extraImages...)
			}

			// For single images, nothing extra to do
		} else if item.Image != nil {
			images = append(images, *item.Image)
		}
	}

	return
}

func (i *API) DownloadImage(image Image) (data []byte, err error) {
	// Imgur actually dislikes Bearer auth on some images
	data, err = i.getUnauthed(image.Link)
	if err != nil {
		data, err = i.get(image.Link)
	}
	return data, err
}

func NewAPI(authUrl string) *API {
	return &API{
		API:            oa2.NewAPI(authUrl),
		unauthedClient: &http.Client{},
	}
}
