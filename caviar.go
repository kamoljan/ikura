package caviar

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"html"
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
	check(err) // real panic
	return b
}

func genPath(eid string, color string, width, height int) string {
	return fmt.Sprintf(caviarStore+"%s/%s/%s", eid[:2], eid[2:4], genFile(eid, color, width, height))
}

func genFile(eid string, color string, width, height int) string {
	return fmt.Sprintf("%04x_%s_%s_%d_%d", caviarId, eid, color, width, height)
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

	imgBaby := resize.Resize(babyWidth, 0, img, resize.NearestNeighbor)
	imgInfant := resize.Resize(infantWidth, 0, imgBaby, resize.NearestNeighbor)
	imgNewborn := resize.Resize(newbornWidth, 0, imgInfant, resize.NearestNeighbor)
	imgSperm := resize.Resize(sperm, sperm, imgNewborn, resize.NearestNeighbor)

	red, green, blue, _ := imgSperm.At(0, 0).RGBA()
	color := fmt.Sprintf("%X%X%X", red>>8, green>>8, blue>>8) // removing 1 byte 9A16->9A

	imgToFile(imgBaby, color)
	imgToFile(imgInfant, color)
	imgToFile(imgNewborn, color)

	t1 := time.Now()
	fmt.Printf("The call took %v to run.\n", t1.Sub(t0))
}

func genHash(img image.Image) string {
	h := sha1.New()
	err := jpeg.Encode(h, img, nil)
	check(err)
	return fmt.Sprintf("%x", h.Sum(nil)) // generate hash
}

func imgToFile(img image.Image, color string) {
	path := genPath(genHash(img), color, img.Bounds().Size().X, img.Bounds().Size().Y) // generate path
	fmt.Println(path)
	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()
	err = jpeg.Encode(out, img, nil) // write image to file
	check(err)
}

func imgToTestFile(img image.Image) {
	path := genFile(genHash(img), "TEST", img.Bounds().Size().X, img.Bounds().Size().Y) + ".jpg"
	fmt.Println(path)
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
	fmt.Println(eid)

	path := parsePath(eid)
	fmt.Println(path)

	http.ServeFile(w, r, path)
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
	initStore("/var/caviar/store")

	http.HandleFunc("/", errorHandler(put))
	http.HandleFunc("/egg/", errorHandler(get))
	http.ListenAndServe(":8080", nil)

	// FIXME: fix (/ad/saved)
	// curl -XPUT http://localhost:8080/ad/saved -H "Content-type: image/jpeg" --data-binary @gopher.png
}
