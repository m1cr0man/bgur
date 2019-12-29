package main

import (
	"flag"
	"fmt"
	"github.com/kirsle/configdir"
	"github.com/m1cr0man/bgur/pkg/bgur"
	"github.com/reujab/wallpaper"
	"os"
	"time"
)

func main() {
	var err error

	folderName := flag.String("folder-name", "desktop backgrounds",
		"Name of the folder to pull desktop backgrounds from")
	folderOwner := flag.String("folder-owner", "",
		"Username who owns the backgrounds folder. Defaults to you")
	expiry := flag.Int("change-interval", 60*12,
		"Minutes between background changes. Default is 12 hours")
	force := flag.Bool("force-change", false,
		"Force a background change now. Overrides expiry")
	refreshCache := flag.Bool("refresh-cache", false,
		"Refresh list of images from the folder on Imgur")
	minRatio := flag.Int("min-ratio", 0,
		"Minimum ratio of width:height, in percent. For example 160 which is 16:10")
	maxRatio := flag.Int("max-ratio", 0,
		"Maximum ratio of width:height, in percent. Use this for vertical screens, overrides minRatio")
	flag.Parse()

	configDir := configdir.LocalConfig("bgur")
	err = configdir.MakePath(configDir) // Ensure it exists.
	if err != nil {
		fmt.Println("Failed to get config dir: ", err)
		os.Exit(1)
		return
	}

	cacheDir := configdir.LocalCache("bgur")
	err = configdir.MakePath(cacheDir) // Ensure it exists.
	if err != nil {
		fmt.Println("Failed to get cache dir: ", err)
		os.Exit(1)
		return
	}

	shutdownChan := make(chan error)

	// Set cache time to 7 days, or refresh now if specified
	cacheTime := time.Hour * 24 * 7
	if *refreshCache {
		cacheTime = 0
	}

	app := bgur.NewApp(configDir, cacheDir, cacheTime)
	go app.RunServer(shutdownChan)

	if err = app.Authorise(); err != nil {
		fmt.Println("Failed to authorise: ", err)
		os.Exit(1)
		return
	}

	if *folderOwner == "" {
		*folderOwner = app.AuthorisedUsername()
	}

	if err = app.SelectFolder(*folderOwner, *folderName); err != nil {
		fmt.Println("Failed to select folder: ", err)
		os.Exit(1)
		return
	}

	if err = app.LoadState(); err != nil {
		fmt.Println("Failed to load state: ", err)
		os.Exit(1)
		return
	}

	fmt.Println("Loading available images")
	if err = app.LoadImages(); err != nil {
		fmt.Println("Failed to load images: ", err)
		os.Exit(1)
		return
	}

	fmt.Println("Picking an image and setting the background")
	if *force {
		*expiry = 0
	}
	image, err := app.PickImage(time.Minute*time.Duration(*expiry), *minRatio, *maxRatio)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
		return
	}
	fmt.Println("Using", image.Link, "as desktop background")

	imagePath, err := app.DownloadImage(image)
	if err != nil {
		fmt.Println("Failed to download image: ", err)
	}

	err = wallpaper.SetFromFile(imagePath)
	if err != nil {
		fmt.Println("Failed to set desktop background: ", err)
	}

	// Save images after picking so that DateSeen is saved
	if err = app.SaveImages(); err != nil {
		fmt.Println("Failed to save cache of images: ", err, " This will slow down subsequent runs")
	}

	if err = app.SaveState(); err != nil {
		fmt.Println("Failed to save state: ", err)
		os.Exit(1)
		return
	}

	// Wait until web app is killed
	app.StopServer()
	_ = <-shutdownChan
}
