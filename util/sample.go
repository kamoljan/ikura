package main

import (
	"image"
	"image/color"
	"os"
	"runtime"

	"github.com/disintegration/imaging"
)

func main() {
	// use all CPU cores for maximum performance
	runtime.GOMAXPROCS(runtime.NumCPU())

	// input files
	files := []string{"1.jpg", "2.jpg", "3.jpg", "4.jpg"}

	// load images and make 100x100 thumbnails of them
	var thumbnails []image.Image
	for _, file := range files {
		img, err := imaging.Open(file)
		if err != nil {
			panic(err)
		}
		thumb := imaging.Thumbnail(img, 100, 100, imaging.CatmullRom)
		thumbnails = append(thumbnails, thumb)
	}

	// create a new blank image
	dst := imaging.New(100*len(thumbnails), 100, color.NRGBA{0, 0, 0, 0})

	f, err := os.OpenFile("db", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	// paste thumbnails into the new image side by side
	for i, thumb := range thumbnails {
		if _, err = f.Write([]byte(thumb)); err != nil {
			panic(err)
		}
		dst = imaging.Paste(dst, thumb, image.Pt(i*100, 0))
	}

	// save the combined image to file
	err = imaging.Save(dst, "dst.jpg")
	if err != nil {
		panic(err)
	}
}
