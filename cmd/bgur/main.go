package main

import (
	"bgur/pkg/imgur"
	"log"
)

func main() {
	api := imgur.ImgurAPI{}

	log.Fatalln(api.Authorize())
}
