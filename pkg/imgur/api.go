package imgur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	oa2 "github.com/m1cr0man/bgur/pkg/oauth2"
	"golang.org/x/oauth2"
)

type API struct {
	*oa2.API
	unauthedClient *http.Client
}

func responseProcessor(res *http.Response, olderr error) (body []byte, err error) {
	if olderr != nil {
		return
	}

	body, err = ioutil.ReadAll(res.Body)
	if res.StatusCode > 299 {
		err = fmt.Errorf("failed to %s %s. Status code %d. Response: %s",
			res.Request.Method, res.Request.URL, res.StatusCode, body)
	}
	return
}

func (i *API) get(url string) (body []byte, err error) {
	return responseProcessor(i.API.Client.Get(url))
}

func (i *API) getUnauthed(url string) (body []byte, err error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return []byte{}, err
	}
	req.Header.Set("User-Agent", "Bgur/0.0.3")
	return responseProcessor(i.unauthedClient.Do(req))
}

func (i *API) post(url string, data url.Values) (body []byte, err error) {
	return responseProcessor(i.API.Client.PostForm(url, data))
}

func (i *API) put(url string) (body []byte, err error) {
	req, err := http.NewRequest(http.MethodPut, url, &bytes.Buffer{})
	if err != nil {
		return
	}
	return responseProcessor(i.API.Client.Do(req))
}

func (i *API) delete(url string) (body []byte, err error) {
	req, err := http.NewRequest(http.MethodDelete, url, &bytes.Buffer{})
	if err != nil {
		return
	}
	return responseProcessor(i.API.Client.Do(req))
}

func (i *API) Authorise(tokenFile string) error {
	i.SetConfig(&oauth2.Config{
		ClientID: "825af7b91a9dfbf",
		Endpoint: oauth2.Endpoint{
			AuthURL:   "https://api.imgur.com/oauth2/authorize",
			TokenURL:  "https://api.imgur.com/oauth2/token",
			AuthStyle: oauth2.AuthStyleInParams,
		},
	})
	return i.API.Authorise(tokenFile)
}

func (i *API) GetAlbums() (albums []Album, err error) {
	var body []byte
	var response AlbumsResponse
	p := 0
	for {
		body, err = i.get(fmt.Sprintf("https://api.imgur.com/3/account/%s/albums/%d", i.API.Username, p))

		if err != nil {
			return
		}

		if err = json.Unmarshal(body, &response); err != nil {
			return
		}

		if len(response.Data) == 0 {
			return
		}

		albums = append(albums, response.Data...)
		p++
	}
}

func (i *API) CreateImage(name, title, description string, albumId string, imgBytes []byte) (image Image, err error) {
	data := url.Values{}

	// Tried using base64 encoded images, they didn't work
	data.Set("name", name)
	data.Set("title", title)
	data.Set("description", description)
	data.Set("image", string(imgBytes))
	data.Set("type", "file")
	if albumId != "" {
		data.Set("album", albumId)
	}

	body, err := i.post("https://api.imgur.com/3/image", data)
	if err != nil {
		return
	}

	var response ImagePostResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}

	image = response.Data
	return
}

func (i *API) UpdateImage(id, title, description string) (err error) {
	data := url.Values{}

	// Tried using base64 encoded images, they didn't work
	data.Set("title", title)
	data.Set("description", description)

	_, err = i.post("https://api.imgur.com/3/image/"+id, data)
	return
}

func (i *API) DeleteImage(imageId string) (err error) {
	body, err := i.delete("https://api.imgur.com/3/image/" + imageId)
	if err != nil {
		return
	}

	var response APIResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}
	if !response.Success {
		return fmt.Errorf("failed to delete image. %v", response.Data)
	}

	return
}

func (i *API) CreateAlbum(title, description string, privacy Privacy, images []Image) (album Album, err error) {
	data := url.Values{}

	data.Set("title", title)
	data.Set("description", description)
	data.Set("privacy", string(privacy))
	for _, image := range images {
		data.Add("ids", image.Id)
	}

	body, err := i.post("https://api.imgur.com/3/album", data)
	if err != nil {
		return
	}

	var response AlbumPostResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return
	}

	album = response.Data
	return
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

func (i *API) AddAlbumToFolder(folderId int, albumId string) (err error) {
	url := fmt.Sprintf("https://api.imgur.com/3/folders/%d/favorites/album/%s", folderId, albumId)
	_, err = i.put(url)
	return
}

func (i *API) DownloadImage(imageLink string) (data []byte, err error) {
	// Imgur actually dislikes Bearer auth on some images
	data, err = i.getUnauthed(imageLink)
	if err != nil {
		data, err = i.get(imageLink)
	}
	return data, err
}

func (i *API) GetFavourites(folderOwner string) (data []ImageOrAlbum, err error) {
	var body []byte
	var response FolderContentResponse
	p := 0
	for {
		body, err = i.get(fmt.Sprintf("https://api.imgur.com/3/account/%s/favorites/%d", folderOwner, p))

		if err != nil {
			return
		}

		if err = json.Unmarshal(body, &response); err != nil {
			return
		}

		if len(response.Data) == 0 {
			return
		}

		data = append(data, response.Data...)
		p++
	}
}

func NewAPI(authUrl string) *API {
	return &API{
		API:            oa2.NewAPI(authUrl),
		unauthedClient: &http.Client{},
	}
}
