package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/kirsle/configdir"
	"github.com/m1cr0man/bgur/pkg/bgur"
)

func main() {
	var err error

	folderName := flag.String("folder-name", "Screenshots",
		"Name of the folder to add new albums to")
	source := flag.String("source", ".",
		"Folder to begin uploading from")
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
	cacheTime := time.Hour * 24 * 7

	app := bgur.NewApp(configDir, cacheDir, cacheTime, false)
	go app.RunServer(shutdownChan)

	if err = app.Authorise(); err != nil {
		fmt.Println("Failed to authorise: ", err)
		os.Exit(1)
		return
	}

	folderOwner := app.AuthorisedUsername()

	if err = app.SelectFolder(folderOwner, *folderName); err != nil {
		fmt.Println("Failed to select folder: ", err)
		os.Exit(1)
		return
	}

	files, err := ioutil.ReadDir(*source)
	if err != nil {
		fmt.Println("Failed to read directory:", err)
		os.Exit(1)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			folderName := file.Name()
			fmt.Println("Uploading directory", folderName)
			err = app.UploadAllImages(*source+"/"+folderName, folderName)
			if err != nil {
				fmt.Println("Failed to upload images to album:", folderName, err)
				os.Exit(1)
				return
			}
		}
	}

	fmt.Println("Finshed uploading images")

	// Wait until web app is killed
	app.StopServer()
	_ = <-shutdownChan
}
