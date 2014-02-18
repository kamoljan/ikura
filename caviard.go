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

func calPath(eid string, width, height int) string {
	//TODO: remove hardcoded configuration to configuration file
	return fmt.Sprintf("/var/caviar/store/%s/%s/%04x_%s_%x_%d_%d", eid[:2], eid[2:4], 1, eid, 0x9a, width, height)
}

func put(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		panic("Not supported Method")
	}
	defer r.Body.Close()
	img, err := jpeg.Decode(r.Body) // decode jpeg into image.Image
	if err != nil {
		log.Fatal(err)
	}
	t0 := time.Now()

	baby_img := resize.Resize(400, 0, img, resize.NearestNeighbor)
	writeToFile(baby_img)
	infant_img := resize.Resize(200, 0, baby_img, resize.NearestNeighbor)
	writeToFile(infant_img)
	newborn_img := resize.Resize(100, 0, infant_img, resize.NearestNeighbor)
	writeToFile(newborn_img)

	t1 := time.Now()
	fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
	//sperm_img := resize.Resize(1, 0, newborn_img, resize.NearestNeighbor)
}

// write image to file
func writeToFile(m image.Image) {
	h := sha1.New()
	err := jpeg.Encode(h, m, nil)
	check(err)
	f := fmt.Sprintf("%x", h.Sum(nil))
	path := calPath(f, m.Bounds().Size().X, m.Bounds().Size().Y)
	fmt.Println(path)

	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	err = jpeg.Encode(out, m, nil)
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
