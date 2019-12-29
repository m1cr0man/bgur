package bgur

import (
	"bgur/pkg/imgur"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// These are hard coded on the Imgur app auth page
const AuthPort = 8099
const AuthUrl = "/oauthcallback"
const CacheTime = time.Hour * 24
const TimeFormat = time.RFC3339

type State struct {
	CurrentImage   int    `json:"current_image"`
	CacheTimestamp string `json:"cache_timestamp"`
	DateChanged    string `json:"date_changed"`
}

type App struct {
	ConfigDir      string
	CacheDir       string
	folderId       int
	api            *imgur.API
	server         *http.Server
	images         []imgur.Image
	currentImage   int
	cacheTimestamp time.Time
	dateChanged    time.Time
}

func (a *App) cacheFile() string {
	return filepath.Join(a.CacheDir, fmt.Sprintf("cache.%d.json", a.folderId))
}

func (a *App) stateFile() string {
	return filepath.Join(a.ConfigDir, fmt.Sprintf("state.%d.json", a.folderId))
}

func (a *App) imageFile(image imgur.Image) string {
	return filepath.Join(a.CacheDir, filepath.Base(image.Link))
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

func (a *App) SelectFolder(folderName string) error {
	folders, err := a.api.GetFolders()
	if err != nil {
		return err
	}

	for _, folder := range folders {
		if strings.ToLower(folder.Name) == strings.ToLower(folderName) {
			a.folderId = folder.Id
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

func (a *App) SaveState() error {
	return a.saveJSON(a.stateFile(), State{
		CurrentImage:   a.currentImage,
		CacheTimestamp: a.cacheTimestamp.Format(TimeFormat),
		DateChanged:    a.dateChanged.Format(TimeFormat),
	})
}

func (a *App) LoadState() error {
	// Try loading the state file
	// Support different folders on the same machine
	stateFile := fmt.Sprintf("%s%cstate.%d.json", a.ConfigDir, os.PathSeparator, a.folderId)
	data, err := ioutil.ReadFile(stateFile)
	var state State
	if err == nil {
		err = json.Unmarshal(data, &state)
		if err != nil {
			return err
		}
	}

	// Copy values into struct and parse them
	a.dateChanged, err = time.Parse(TimeFormat, state.DateChanged)
	if err != nil && state.DateChanged != "" {
		return err
	}
	a.cacheTimestamp, err = time.Parse(TimeFormat, state.CacheTimestamp)
	if err != nil && state.CacheTimestamp != "" {
		return err
	}
	a.currentImage = state.CurrentImage
	return nil
}

func (a *App) LoadImages() (err error) {
	var newImages []imgur.Image

	// Try loading the folder cache
	data, err := ioutil.ReadFile(a.cacheFile())
	err2 := json.Unmarshal(data, &a.images)
	expired := a.cacheTimestamp.Add(CacheTime).Before(time.Now())

	// Any errors with the cache can be ignored, we can rebuild it
	if err != nil || err2 != nil || expired {
		newImages, err = a.api.GetFolderImages(a.folderId)
		Randomise(newImages)
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

func (a *App) PickImage(expiry time.Duration) imgur.Image {

	// Select currentImage if it has not expired
	currentImage := a.currentImage
	if a.dateChanged.Add(expiry).After(time.Now()) {
		return a.images[currentImage]
	}

	// Increment currentImage
	currentImage++
	if currentImage == len(a.images) {
		currentImage = 0
	}

	// Select new image
	a.currentImage = currentImage
	a.dateChanged = time.Now()
	return a.images[currentImage]
}

func (a *App) DownloadImage(image imgur.Image) (imgPath string, err error) {
	imgPath = a.imageFile(image)

	// Check image already exists
	if _, err = os.Stat(imgPath); err == nil {
		return
	}

	imgData, err := a.api.DownloadImage(image)

	if err != nil {
		return
	}

	err = ioutil.WriteFile(imgPath, imgData, 0644)
	return
}

func NewApp(configDir, cacheDir string) *App {
	return &App{
		ConfigDir: configDir,
		CacheDir:  cacheDir,
		server:    &http.Server{Addr: fmt.Sprintf(":%d", AuthPort)},
		api:       imgur.NewAPI(AuthUrl),
	}
}
