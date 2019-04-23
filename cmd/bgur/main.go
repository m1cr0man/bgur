package main

import (
	"bgur/pkg/imgur"
	"fmt"
	"github.com/kirsle/configdir"
	"os"
	"path/filepath"
)

func main() {
	api := imgur.ImgurAPI{}

	configPath := configdir.LocalConfig("bgur")
	err := configdir.MakePath(configPath) // Ensure it exists.
	if err != nil {
		fmt.Println("Failed to get config dir: ", err)
		os.Exit(1)
		return
	}

	authFile := filepath.Join(configPath, "token.json")
	err = api.Authorize(authFile)
	if err != nil {
		fmt.Println("Failed to authorise: ", err)
		os.Exit(1)
		return
	}
}
