package bgur

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/m1cr0man/bgur/pkg/imgur"
)

// These are hard coded on the Imgur app auth page
const AuthPort = 8099
const AuthUrl = "/oauthcallback"

type App struct {
	parsedState
	ConfigDir   string
	CacheDir    string
	CacheTime   time.Duration
	Sync        bool
	folderOwner string
	folderId    int
	api         *imgur.API
	server      *http.Server
	images      []imgur.Image
	stateAlbum  imgur.Album
	stateImage  imgur.Image
}

func (a *App) cacheFile() string {
	return filepath.Join(a.CacheDir, fmt.Sprintf("cache.%s.%d.json", a.folderOwner, a.folderId))
}

func (a *App) CountImages() int {
	return len(a.images)
}

func (a *App) imageFile(image imgur.Image) string {
	return filepath.Join(a.CacheDir, filepath.Base(image.Link))
}

func (a *App) AuthorisedUsername() string {
	return a.api.Username
}

func (a *App) RunServer(shutdownChan chan error) {
	shutdownChan <- a.server.ListenAndServe()
}

func (a *App) StopServer() {
	_ = a.server.Shutdown(context.Background())
}

func (a *App) Authorise() error {
	authFile := filepath.Join(a.ConfigDir, "token.json")
	return a.api.Authorise(authFile)
}

func (a *App) SelectFolder(folderOwner, folderName string) error {
	folders, err := a.api.GetFolders(folderOwner)
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if strings.ToLower(folder.Name) == strings.ToLower(folderName) {
			a.folderId = folder.Id
			a.folderOwner = folderOwner
			return nil
		}
	}

	options := make([]string, len(folders))
	for i, folder := range folders {
		options[i] = fmt.Sprintf("\t- %s", folder.Name)
	}
	return fmt.Errorf("could not find a folder called %s. Options:\n" + strings.Join(options, "\n"))
}

func (a *App) saveJSON(filePath string, data interface{}) error {
	marshalled, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, marshalled, 0644)
}

func (a *App) SaveImages() error {
	return a.saveJSON(a.cacheFile(), a.images)
}

func (a *App) LoadImages() (err error) {
	var newImages []imgur.Image

	// Try loading the folder cache
	data, err := ioutil.ReadFile(a.cacheFile())
	err2 := json.Unmarshal(data, &a.images)
	expired := a.cacheTimestamp.Add(a.CacheTime).Before(time.Now())

	// Any errors with the cache can be ignored, we can rebuild it
	if err != nil || err2 != nil || expired {
		newImages, err = a.api.GetFolderImages(a.folderOwner, a.folderId)
		if a.seed > 0 {
			rand.Seed(a.seed)
			Randomise(newImages)
		}
		if err != nil {
			return err
		}

		a.cacheTimestamp = time.Now()

	} else {
		// No errors means we should use the old list
		newImages = a.images
	}

	// If we have old images, preserve the list of seen images
	if a.currentImage > 0 && expired {

		// Identify all the images we've seen before, preserve them in a separate list
		seen := make(map[string]imgur.Image, a.currentImage)
		for _, image := range a.images[:a.currentImage] {
			seen[image.Id] = image
		}

		// Move the seen images to the start of the list
		var i int
		for j, image := range newImages {
			if _, found := seen[image.Id]; found {
				newImages[i], newImages[j] = newImages[j], newImages[i]
				i++
			}
		}

		// If some seen images were removed, fix the offset of CurrentImage
		a.currentImage -= a.currentImage - i
	}

	a.images = newImages
	return
}

func (a *App) PickImage(expiry time.Duration, minRatio, maxRatio int) (imgur.Image, error) {

	// Select currentImage if it has not expired
	currentImage := a.currentImage
	if a.dateChanged.Add(expiry).After(time.Now()) {
		return a.images[currentImage], nil
	}

	// Loop until an appropriate image is found
	for i := 0; i < len(a.images); i++ {
		// Increment currentImage
		currentImage++
		if currentImage == len(a.images) {
			currentImage = 0
		}

		newImage := a.images[currentImage]

		// Check image MIME and skip animated images
		if newImage.Animated || !strings.Contains(newImage.Type, "image") {
			continue
		}

		// Check ratio, skip to next image if wrong
		if (minRatio > 0 && newImage.Ratio() < minRatio) || (maxRatio > 0 && newImage.Ratio() > maxRatio) {
			continue
		}

		// Select new image
		a.currentImage = currentImage
		a.dateChanged = time.Now()
		return newImage, nil
	}

	// If the loop exits, then no images matched the filter. Return the remaining currentImage
	return a.images[currentImage], fmt.Errorf("no new image found. Perhaps filters are too strict")
}

func (a *App) DownloadImage(image imgur.Image) (imgPath string, err error) {
	imgPath = a.imageFile(image)

	// Check image already exists
	if _, err = os.Stat(imgPath); err == nil {
		return
	}

	imgData, err := a.api.DownloadImage(image.Link)

	if err != nil {
		return
	}

	err = ioutil.WriteFile(imgPath, imgData, 0644)
	return
}

func (a *App) SetSeed(seed int64) {
	// If we've already generated a random seed don't change it
	if seed == -1 && a.seed < 1 {
		a.seed = seed
	}
}

func (a *App) DumpFavourites(folderOwner string) (err error) {
	data, err := a.api.GetFavourites(folderOwner)
	if err != nil {
		return
	}
	js, err := json.Marshal(data)
	if err != nil {
		return
	}
	return ioutil.WriteFile("favourites.json", js, 0440)
}

func (a *App) UploadAllImages(sourcePath, albumName string) (err error) {
	var f os.FileInfo
	var files []os.FileInfo
	var image imgur.Image
	var images []imgur.Image
	var content []byte
	var sleepTime time.Duration = 1

	files, err = ioutil.ReadDir(sourcePath)
	if err != nil {
		return
	}
	for _, f = range files {
		if !f.IsDir() {
			content, err = ioutil.ReadFile(path.Join(sourcePath, f.Name()))
			fmt.Println("Uploading ", f.Name())
			image, err = a.api.CreateImage(f.Name(), "", "Uploaded from bgur", "", content)
			if err != nil {
				if strings.Contains(err.Error(), "too fast") {
					print("Getting rate limited! Sleeping longer between uploads")
					sleepTime *= 2
					if sleepTime > 5 {
						sleepTime = 5
					}
					time.Sleep(time.Minute * sleepTime)
					image, err = a.api.CreateImage(f.Name(), "", "Uploaded from bgur", "", content)
					if err != nil {
						return
					}
				} else {
					return
				}
			}
			images = append(images, image)
			fmt.Printf("Uploaded %s to %s\n", f.Name(), image.Link)
		}
	}

	album, err := a.api.CreateAlbum(albumName, "Uploaded from bgur", imgur.PrivacyHidden, images)
	if err != nil {
		return
	}
	fmt.Printf("Created album: %s\n", album.Link)
	return
}

func (a *App) AddAlbumToFolder(albumId string) (err error) {
	return a.api.AddAlbumToFolder(a.folderId, albumId)
}

func NewApp(configDir, cacheDir string, cacheTime time.Duration, sync bool) *App {
	return &App{
		ConfigDir: configDir,
		CacheDir:  cacheDir,
		CacheTime: cacheTime,
		Sync:      sync,
		server:    &http.Server{Addr: fmt.Sprintf(":%d", AuthPort)},
		api:       imgur.NewAPI(AuthUrl),
	}
}
