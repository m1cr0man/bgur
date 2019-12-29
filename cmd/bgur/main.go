package main

import (
	"bgur/pkg/bgur"
	"bgur/pkg/desktop"
	"flag"
	"fmt"
	"github.com/kirsle/configdir"
	"os"
	"time"
)

func main() {
	var err error

	folderName := flag.String("folder-name", "desktop backgrounds",
		"Name of the folder to pull desktop backgrounds from")
	folderOwner := flag.String("folder-owner", "",
		"Username who owns the backgrounds folder. Defaults to you")
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
	app := bgur.NewApp(configDir, cacheDir)
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

	// TODO stop here if image is already downloaded? and skip auth?

	fmt.Println("Loading available images")
	if err = app.LoadImages(); err != nil {
		fmt.Println("Failed to load images: ", err)
		os.Exit(1)
		return
	}

	fmt.Println("Picking an image and setting the background")
	// TODO Arg for expiry + force argument
	// TODO filters (aspect ratio)
	image := app.PickImage(time.Hour * 24)

	imagePath, err := app.DownloadImage(image)
	if err != nil {
		fmt.Println("Failed to download image: ", err)
	}

	err = desktop.SetBackground(imagePath)
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
