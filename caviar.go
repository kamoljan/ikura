package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html"
	"image"
	"image/jpeg"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/nfnt/resize"
)

//TODO: move it into conf file
const (
	caviarId    = 1
	caviarStore = "/var/caviar/store/"

	babyWidth    = 400
	infantWidth  = 200
	newbornWidth = 100
	sperm        = 1

	cacheMaxAge = 30 * 24 * 60 * 60 // 30 days
	mime        = "image/jpeg"
)

type Eggs struct {
	Origin, Baby, Infant, Newborn string
}

type Msg struct {
	Status string
	Result interface{}
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

func Message(status string, message interface{}) []byte {
	m := Msg{
		Status: status,
		Result: message,
	}
	b, err := json.Marshal(m)
	check(err) // real panic
	return b
}

func genPath(file string) string {
	path := fmt.Sprintf(caviarStore+"%s/%s/%s", file[5:7], file[7:9], file)
	fmt.Println(path)
	return path
}

func genFile(eid string, color string, width, height int) string {
	file := fmt.Sprintf("%04x_%s_%s_%d_%d", caviarId, eid, color, width, height)
	fmt.Println(file)
	return file
}

func put(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		panic("Not supported Method")
	}

	fmt.Println(r)

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	buf := bytes.NewBufferString("")
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if part.FileName() == "" { // if empy skip this iteration
			continue
		}
		_, err = io.Copy(buf, part)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	defer r.Body.Close()
	img, _, err := image.Decode(buf)
	if err != nil {
		log.Panic(err) // log.Fatal(err)
	}
	t0 := time.Now()

	imgBaby := resize.Resize(babyWidth, 0, img, resize.NearestNeighbor)
	imgInfant := resize.Resize(infantWidth, 0, imgBaby, resize.NearestNeighbor)
	imgNewborn := resize.Resize(newbornWidth, 0, imgInfant, resize.NearestNeighbor)
	imgSperm := resize.Resize(sperm, sperm, imgNewborn, resize.NearestNeighbor)

	red, green, blue, _ := imgSperm.At(0, 0).RGBA()
	color := fmt.Sprintf("%X%X%X", red>>8, green>>8, blue>>8) // removing 1 byte 9A16->9A

	fileOrig := imgToFile(img, color)
	fileBaby := imgToFile(imgBaby, color)
	fileInfant := imgToFile(imgInfant, color)
	fileNewborn := imgToFile(imgNewborn, color)

	result := Eggs{
		Origin:  fileOrig,
		Baby:    fileBaby,
		Infant:  fileInfant,
		Newborn: fileNewborn,
	}

	if err != nil {
		w.Write(Message("ERROR", "Was not able to save your file"))
	} else {
		w.Write(Message("OK", &result))
	}

	t1 := time.Now()
	fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
}

// func putOld(w http.ResponseWriter, r *http.Request) {
// 	if r.Method != "PUT" {
// 		panic("Not supported Method")
// 	}

// 	fmt.Println(r)

// 	defer r.Body.Close()
// 	img, _, err := image.Decode(r.Body)
// 	if err != nil {
// 		log.Panic(err) // log.Fatal(err)
// 	}
// 	t0 := time.Now()

// 	imgBaby := resize.Resize(babyWidth, 0, img, resize.NearestNeighbor)
// 	imgInfant := resize.Resize(infantWidth, 0, imgBaby, resize.NearestNeighbor)
// 	imgNewborn := resize.Resize(newbornWidth, 0, imgInfant, resize.NearestNeighbor)
// 	imgSperm := resize.Resize(sperm, sperm, imgNewborn, resize.NearestNeighbor)

// 	red, green, blue, _ := imgSperm.At(0, 0).RGBA()
// 	color := fmt.Sprintf("%X%X%X", red>>8, green>>8, blue>>8) // removing 1 byte 9A16->9A

// 	fileBaby := imgToFile(imgBaby, color)
// 	fileInfant := imgToFile(imgInfant, color)
// 	fileNewborn := imgToFile(imgNewborn, color)

// 	result := Eggs{
// 		baby:    fileBaby,
// 		infant:  fileInfant,
// 		newborn: fileNewborn,
// 	}

// 	if err != nil {
// 		w.Write(Message("ERROR", "Was not able to save your file"))
// 	} else {
// 		w.Write(Message("OK", result))
// 	}

// 	t1 := time.Now()
// 	fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
// }

func genHash(img image.Image) string {
	h := sha1.New()
	err := jpeg.Encode(h, img, nil)
	check(err)
	return fmt.Sprintf("%x", h.Sum(nil)) // generate hash
}

func imgToFile(img image.Image, color string) string {
	file := genFile(genHash(img), color, img.Bounds().Size().X, img.Bounds().Size().Y)
	path := genPath(file)
	out, err := os.Create(path)
	check(err)
	defer out.Close()
	err = jpeg.Encode(out, img, nil) // write image to file
	check(err)
	return file
}

func imgToTestFile(img image.Image) {
	path := genFile(genHash(img), "TEST", img.Bounds().Size().X, img.Bounds().Size().Y) + ".jpg"
	out, err := os.Create(path)
	check(err)
	defer out.Close()
	err = jpeg.Encode(out, img, nil) // write image to file
	check(err)
}

func parsePath(eid string) string {
	return fmt.Sprintf(caviarStore+"%s/%s/%s", eid[5:7], eid[7:9], eid)
}

func get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", cacheMaxAge))
	// TODO: need to add Expires into Header
	// self.set_common_header()
	// self.set_header("Content-Type", self.application.options["mime"])
	// self.set_header("Cache-Control", "max-age="+str(CACHE_MAX_AGE))
	// self.set_header("Expires", datetime.datetime.utcfromtimestamp(info.st_mtime+CACHE_MAX_AGE))
	// self.set_header("Last-Modified", datetime.datetime.utcfromtimestamp(info.st_mtime))
	// self.set_header("Date", datetime.datetime.utcfromtimestamp(time.time()))
	eid := html.EscapeString(r.URL.Path[5:])
	path := parsePath(eid)
	http.ServeFile(w, r, path)
}

func initStore(path string) {
	fmt.Println("Initializing data store...")
	for i := 0; i < 256; i++ {
		for x := 0; x < 256; x++ {
			err := os.MkdirAll(fmt.Sprintf("%s/%02x/%02x", path, i, x), 0755)
			check(err)
		}
	}
	fmt.Println("...Done") // total 65536 directories
}

func main() {
	initStore(caviarStore)
	http.HandleFunc("/", errorHandler(put))
	http.HandleFunc("/egg/", errorHandler(get))
	http.ListenAndServe(":9090", nil)

	// FIXME: fix (/ad/saved)
	// curl -XPUT http://localhost:8080/ad/saved -H "Content-type: image/jpeg" --data-binary @gopher.png
}
