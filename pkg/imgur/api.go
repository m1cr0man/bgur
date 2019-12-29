package imgur

import (
	oa2 "bgur/pkg/oauth2"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io/ioutil"
)

type API struct {
	*oa2.API
}

func (i *API) get(url string) (body []byte, err error) {
	res, err := i.API.Client.Get(url)
	if err != nil {
		return
	}

	return ioutil.ReadAll(res.Body)
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

func (i *API) GetFolders() (folders []Folder, err error) {
	body, err := i.get("https://api.imgur.com/3/folders")
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

func (i *API) GetFolderImages(folderId int) (images []Image, err error) {
	body, err := i.get(fmt.Sprintf("https://api.imgur.com/3/folders/%d/favorites", folderId))
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
	return i.get(image.Link)
}

func NewAPI(authUrl string) *API {
	return &API{
		API: oa2.NewAPI(authUrl),
	}
}
