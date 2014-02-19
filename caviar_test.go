package main

import (
	"fmt"
	"image"
	"net/http"
	"testing"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

func TestGet(t *testing.T) {
	// /var/caviar/store/58/ba/0001_58baa4e5acbaafcb60c260b1dd61e4feb26e986e_9AA2A8_400_300
	r, err := http.Get("http://localhost:8080/egg/0001_1782d92a5815a4692cc1b37fc1e145c6c90dbb3b_9BA4A9_400_300")
	check(err)
	img, img_format, err := image.Decode(r.Body) // decode jpeg into image.Image
	defer r.Body.Close()
	fmt.Println(img_format)
	check(err)
	fmt.Println(genHash(img))

	imgToTestFile(img)
	check(err)
}
