package main

import (
	"fmt"
	"image/jpeg"
	"log"
	"os"
	// Package image/jpeg is not used explicitly in the code below,
	// but is imported for its initialization side-effect, which allows
	// image.Decode to understand JPEG formatted images. Uncomment these
	// two lines to also understand GIF and PNG images:
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/nfnt/resize"
)

func main() {
	// open "1.jpg"
	file, err := os.Open("3.jpg")
	if err != nil {
		log.Fatal(err)
	}

	// decode jpeg into image.Image
	img, err := jpeg.Decode(file)
	if err != nil {
		log.Fatal(err)
	}
	file.Close()

	// resize to width 1000 using Lanczos resampling
	// and preserve aspect ratio
	m := resize.Resize(1, 0, img, resize.NearestNeighbor)

	r, g, b, _ := m.At(0, 0).RGBA()

	fmt.Printf("%X%X%X", r>>8, g>>8, b>>8) // removing 1 byte 9A16->9A

	out, err := os.Create("test_resized3.jpg")
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// write new image to file
	jpeg.Encode(out, m, nil)
}
