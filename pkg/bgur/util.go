package bgur

import (
	"bgur/pkg/imgur"
	"math/rand"
	"time"
)

// Returns elements in A not in B
func simpleDiff(sliceA, sliceB []imgur.Image) (diff []imgur.Image) {
	imgMap := make(map[string]imgur.Image, len(sliceB))

	for _, image := range sliceB {
		imgMap[image.Id] = image
	}

	for _, image := range sliceA {
		if _, found := imgMap[image.Id]; !found {
			diff = append(diff, image)
		}
	}
	return
}

// Returns elements in A not in B, and vice versa
func DiffImages(sliceA, sliceB []imgur.Image) ([]imgur.Image, []imgur.Image) {
	return simpleDiff(sliceA, sliceB), simpleDiff(sliceB, sliceA)
}

func Randomise(images []imgur.Image) {
	rand.Seed(time.Now().Unix())
	rand.Shuffle(len(images), func(i, j int) { images[i], images[j] = images[j], images[i] })
}
