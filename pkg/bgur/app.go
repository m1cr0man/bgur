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
	albums      []imgur.Album
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
	return fmt.Errorf("could not find a folder called %s. Options:\n"+strings.Join(options, "\n"), folderName)
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
	// TODO what if 2 syncing machines update thier cache at the same time?
	// There's no way to know if currentImage should be updated because we may
	// be behind. Might need to store a tuple of (hash, pos) in the sync data.
	// OR we denote position by the image id??
	if a.currentImage > 0 && expired && len(a.images) > a.currentImage {

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

func (a *App) LoadAlbums() (err error) {
	if len(a.albums) > 0 {
		return
	}

	albums, err := a.api.GetAlbums()
	if err != nil {
		return
	}

	a.albums = albums
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
	minWait := time.Second * 30
	maxWait := time.Minute * 10
	sleepTime := minWait
	var f os.FileInfo
	var files []os.FileInfo
	var image imgur.Image
	var existingImage imgur.Image
	var images []imgur.Image
	var existingImages []imgur.Image
	var content []byte
	var album imgur.Album
	var existingAlbum bool

	// Make this idempotent. Load list of images from the album if it already exists
	a.LoadAlbums()

	for _, album = range a.albums {
		if strings.ToLower(album.Title) == strings.ToLower(albumName) {
			existingAlbum = true
			break
		}
	}

	if existingAlbum {
		existingImages, err = a.api.GetAlbumImages(album.Id)
		if err != nil {
			return
		}
	} else {
		// Create a new album if necessary
		album, err = a.api.CreateAlbum(albumName, "Uploaded from bgur", imgur.PrivacyHidden, images)
		if err != nil {
			return
		}
		a.albums = append(a.albums, album)
		fmt.Printf("Created album: %s\n", album.Link)
		err = a.AddAlbumToFolder(album.Id)
		if err != nil {
			return
		}
	}

	files, err = ioutil.ReadDir(sourcePath)
	if err != nil {
		return
	}
	for _, f = range files {
		if !f.IsDir() {
			fname := f.Name()
			title := fname
			description := "Uploaded from bgur. Original date: " + f.ModTime().Format("Jan 2 15:04:05 2006")

			// Skip existing images
			var found bool
			for _, existingImage = range existingImages {
				ename := strings.ToLower(existingImage.Name)
				extStart := strings.LastIndex(fname, ".")
				if strings.ToLower(fname) == ename || strings.ToLower(fname[:extStart]) == ename {
					found = true
					break
				}
			}

			// Found + no update necessary
			if found && existingImage.Title != "" {
				continue
			}

			content, err = ioutil.ReadFile(path.Join(sourcePath, fname))
			for {
				if found {
					fmt.Println("Updating info for", fname)
					err = a.api.UpdateImage(existingImage.Id, title, description)
					time.Sleep(sleepTime / 60)
				} else {
					fmt.Println("Uploading ", fname)
					time.Sleep(sleepTime)
					image, err = a.api.CreateImage(
						fname,
						title,
						description,
						album.Id,
						content,
					)
				}
				if err != nil {
					if strings.Contains(err.Error(), "too fast") {
						sleepTime *= 2
						if sleepTime > maxWait {
							sleepTime = maxWait
						}
						fmt.Println(
							"Getting rate limited! Sleeping",
							sleepTime,
							"between uploads",
						)
						fmt.Println(err.Error())
						continue
					}
					return
				}
				sleepTime -= time.Minute
				if sleepTime < minWait {
					sleepTime = minWait
				}
				break
			}
			if !found {
				images = append(images, image)
				fmt.Printf("Uploaded %s to %s\n", fname, image.Link)
			}
		}
	}

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
