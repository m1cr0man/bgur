package bgur

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/m1cr0man/bgur/pkg/imgur"
	"github.com/makiuchi-d/gozxing"
	"github.com/makiuchi-d/gozxing/qrcode"
	imgLib "image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io/ioutil"
	"path/filepath"
	"time"
)

const StateAlbumName = "Bgur Sync Data"
const TimeFormat = time.RFC3339

type State struct {
	CurrentImage int `json:"current_image"`
	// TODO load cacheTimestamp from cache file, remove from state
	CacheTimestamp string `json:"cache_timestamp"`
	DateChanged    string `json:"date_changed"`
	StateTimestamp string `json:"state_timestamp"`
	Seed           int64  `json:"seed"`
}

type parsedState struct {
	currentImage   int
	cacheTimestamp time.Time
	dateChanged    time.Time
	stateTimestamp time.Time
	seed           int64
}

func (a *App) getState() State {
	return State{
		CurrentImage:   a.currentImage,
		CacheTimestamp: a.cacheTimestamp.Format(TimeFormat),
		DateChanged:    a.dateChanged.Format(TimeFormat),
		StateTimestamp: time.Now().Format(TimeFormat),
		Seed:           a.seed,
	}
}

func (a *App) parseRawState(data []byte) (parsedState parsedState, err error) {
	var state State
	err = json.Unmarshal(data, &state)
	if err != nil {
		return
	}

	// Copy values into struct and parse them
	parsedState.dateChanged, err = time.Parse(TimeFormat, state.DateChanged)
	// Only fail if the date changed wasn't nil to begin with
	if err != nil && state.DateChanged != "" {
		return
	}
	parsedState.cacheTimestamp, err = time.Parse(TimeFormat, state.CacheTimestamp)
	if err != nil && state.CacheTimestamp != "" {
		return
	}
	parsedState.stateTimestamp, err = time.Parse(TimeFormat, state.StateTimestamp)
	if err != nil && state.StateTimestamp != "" {
		return
	}

	// Make sure err is nil after parsing all timestamps
	err = nil

	parsedState.currentImage = state.CurrentImage
	parsedState.seed = state.Seed
	return
}

func (a *App) stateId() string {
	return fmt.Sprintf("%s.%d", a.folderOwner, a.folderId)
}

func (a *App) stateFile() string {
	return filepath.Join(a.ConfigDir, fmt.Sprintf("state.%s.json", a.stateId()))
}

func (a *App) GetStateAlbum() (album imgur.Album, err error) {
	if a.stateAlbum.Id != "" {
		return a.stateAlbum, nil
	}

	albums, err := a.api.GetAlbums()
	if err != nil {
		return
	}

	for _, album = range albums {
		if album.Title == StateAlbumName {
			a.stateAlbum = album
			return
		}
	}

	// Create the album
	return a.api.CreateAlbum(StateAlbumName, "Created automatically by Bgur."+
		" Holds state for syncing backgrounds across computers", imgur.PrivacyHidden, []imgur.Image{})
}

func (a *App) GetStateImage() (image imgur.Image, err error) {
	if a.stateImage.Id != "" {
		return a.stateImage, nil
	}

	album, err := a.GetStateAlbum()
	if err != nil {
		return
	}

	images, err := a.api.GetAlbumImages(album.Id)
	if err != nil {
		return
	}

	stateId := a.stateId()
	for _, image = range images {
		if image.Title == stateId {
			a.stateImage = image
			return
		}
	}

	return
}

func (a *App) UploadState(state State) (err error) {
	image, err := a.GetStateImage()
	if err != nil {
		return
	}

	// Delete old state
	if image.Id != "" {
		err = a.api.DeleteImage(image.Id)
		if err != nil {
			return
		}
	}

	jsonState, err := json.Marshal(state)
	if err != nil {
		return
	}

	imgBytes := &bytes.Buffer{}
	qrCode := qrcode.NewQRCodeWriter()
	bitmap, err := qrCode.EncodeWithoutHint(string(jsonState), gozxing.BarcodeFormat_QR_CODE, 512, 512)
	if err != nil {
		return
	}

	err = png.Encode(imgBytes, bitmap)
	if err != nil {
		return
	}

	a.stateImage, err = a.api.CreateImage("state.png", a.stateId(),
		"Last updated on "+time.Now().Format(time.RFC1123), a.stateAlbum.Id, imgBytes.Bytes())
	return
}

func (a *App) DownloadState() (data []byte, err error) {
	image, err := a.GetStateImage()
	if err != nil {
		return
	}

	// No image, return nil
	if image.Id == "" {
		return
	}

	imgData, err := a.api.DownloadImage(image.Link)
	if err != nil {
		return
	}

	imgDecoded, _, err := imgLib.Decode(bytes.NewReader(imgData))
	if err != nil {
		return
	}

	imgBitmap, err := gozxing.NewBinaryBitmapFromImage(imgDecoded)
	if err != nil {
		return
	}

	reader := qrcode.NewQRCodeReader()
	result, err := reader.DecodeWithoutHints(imgBitmap)
	if err != nil {
		return
	}

	return []byte(result.GetText()), nil
}

func (a *App) SaveState() (err error) {
	a.stateTimestamp = time.Now()
	state := a.getState()
	err = a.saveJSON(a.stateFile(), state)
	if err != nil || !a.Sync {
		return
	}

	// Sync with imgur
	return a.UploadState(state)
}

func (a *App) LoadState() (err error) {
	// Try loading the state file
	// Support different folders on the same machine
	data, err := ioutil.ReadFile(a.stateFile())
	if err == nil {
		a.parsedState, err = a.parseRawState(data)
		if err != nil {
			return
		}
	}

	// Sync with imgur
	if a.Sync {
		data, err = a.DownloadState()
		if err != nil {
			return
		}

		if len(data) > 0 {
			var downloadedState parsedState
			downloadedState, err = a.parseRawState(data)
			if err != nil {
				return
			}

			if downloadedState.stateTimestamp.After(a.stateTimestamp) {
				a.parsedState = downloadedState
			}
		}
	}

	// Conditional err above, make sure it is nil now
	return nil
}
