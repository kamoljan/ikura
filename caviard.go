package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"os"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/nfnt/resize"
)

type Msg struct {
	Status, Message string
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func errorHandler(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(500)
				w.Write(Message("ERROR", err.(string)))
			}
		}()
		fn(w, r)
	}
}

func Message(status string, message string) []byte {
	m := Msg{
		Status:  status,
		Message: message,
	}
	b, err := json.Marshal(m)
	if err != nil {
		panic(err) // real panic
	}
	return b
}

func genPath(eid string, color string, width, height int) string {
	//TODO: remove hardcoded configuration to configuration file
	return fmt.Sprintf("/var/caviar/store/%s/%s/%04x_%s_%s_%d_%d", eid[:2], eid[2:4], 1, eid, color, width, height)
}

func put(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		panic("Not supported Method")
	}
	defer r.Body.Close()
	img, img_format, err := image.Decode(r.Body) // decode jpeg into image.Image
	fmt.Println(img_format)
	if err != nil {
		log.Fatal(err)
	}
	t0 := time.Now()

	img_baby := resize.Resize(400, 0, img, resize.NearestNeighbor)
	img_infant := resize.Resize(200, 0, img_baby, resize.NearestNeighbor)
	img_newborn := resize.Resize(100, 0, img_infant, resize.NearestNeighbor)
	img_sperm := resize.Resize(1, 0, img_newborn, resize.NearestNeighbor)

	red, green, blue, _ := img_sperm.At(0, 0).RGBA()
	color := fmt.Sprintf("%X%X%X", red>>8, green>>8, blue>>8) // removing 1 byte 9A16->9A

	imgToFile(img_baby, color)
	imgToFile(img_infant, color)
	imgToFile(img_newborn, color)

	t1 := time.Now()
	fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
}

// write image to file
func imgToFile(img image.Image, color string) {
	h := sha1.New()
	err := jpeg.Encode(h, img, nil)
	check(err)
	f := fmt.Sprintf("%x", h.Sum(nil)) // generate hash

	path := genPath(f, color, img.Bounds().Size().X, img.Bounds().Size().Y) // generate path
	fmt.Println(path)

	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	err = jpeg.Encode(out, img, nil) // write image to file
	check(err)
}

func view(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image")
	http.ServeFile(w, r, "image-"+r.FormValue("id"))
}

func initStore(path string) {
	fmt.Println("Initializing data store...")
	for i := 0; i < 256; i++ {
		for x := 0; x < 256; x++ {
			err := os.MkdirAll(fmt.Sprintf("%s/%02x/%02x", path, i, x), 0755)
			if err != nil {
				panic(err)
			}
		}
	}
	fmt.Println("...Done") // total 65536 directories
}

func main() {
	// initialize data store
	initStore("/var/caviar/store")

	http.HandleFunc("/", errorHandler(put))
	http.HandleFunc("/view", errorHandler(view))
	http.ListenAndServe(":8080", nil)

	// TEST
	// curl -XPUT http://localhost:8080/ad/saved -H "Content-type: image/jpeg" --data-binary @gopher.png
}
