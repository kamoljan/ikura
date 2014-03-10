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

	"labix.org/v2/mgo"

	"github.com/nfnt/resize"
)

const ( // TODO: Move to Conf file
	ikuraId    = 1
	ikuraStore = "/var/ikura/store/"

	babyWidth    = 400
	infantWidth  = 200
	newbornWidth = 100
	sperm        = 1

	cacheMaxAge = 30 * 24 * 60 * 60 // 30 days
	mime        = "image/jpeg"

	mongodb = "mongodb://admin:12345678@localhost:27017/sa"
)

type Egg struct {
	Egg     string `json:"egg"`     //0001_bbf06d39e4dac6b4cac5ee16226f6b5f7c50f071_ACA0AC_401_638
	Baby    string `json:"baby"`    //0001_6881db255b21c864c9d1e28db50dc3b71dab5b78_ACA0AC_400_637
	Infant  string `json:"infant"`  //0001_ff41e42b0134e219bc09eddda87687822460afcf_ACA0AC_200_319
	Newborn string `json:"newborn"` //0001_040db0bc2fc49ab41fd81294c7d195c7d1de358b_ACA0AC_100_160
}

type Result struct {
	Egg string `json:"egg"` //0001_bbf06d39e4dac6b4cac5ee16226f6b5f7c50f071_ACA0AC_401_638
}

type Msg struct {
	Status string      `json:"status"` //"ok"
	Result interface{} `json:"result"` //{egg: "0001_bbf06d39e4dac6b4cac5ee16226f6b5f7c50f071_ACA0AC_401_638"}
}

func (egg *Egg) saveMeta() error {
	session, err := mgo.Dial(mongodb)
	if err != nil {
		log.Fatal("Was not able to connect to DB ", err)
	}

	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("sa").C("egg")
	err = c.Insert(&egg)
	if err != nil {
		log.Fatal("Was not able to save to DB ", err)
	}
	return err
}

func Message(status string, message interface{}) []byte {
	m := Msg{
		Status: status,
		Result: message,
	}
	b, err := json.Marshal(m)
	if err != nil {
		log.Println("Was not able to json.Marshal ", err)
	}
	return b
}

func genPath(file string) string {
	path := fmt.Sprintf(ikuraStore+"%s/%s/%s", file[5:7], file[7:9], file)
	log.Println(path)
	return path
}

func genFile(eid string, color string, width, height int) string {
	file := fmt.Sprintf("%04x_%s_%s_%d_%d", ikuraId, eid, color, width, height)
	log.Println(file)
	return file
}

/*
 *{
 *	status: "ok"
 * 	result: { egg: "0001_bbf06d39e4dac6b4cac5ee16226f6b5f7c50f071_ACA0AC_401_638" }
 *}
 */
func put(w http.ResponseWriter, r *http.Request) {
	if r.Method != "PUT" {
		w.Write(Message("ERROR", "Not supported Method"))
		return
	}

	log.Println(r)

	reader, err := r.MultipartReader()
	if err != nil {
		w.Write(Message("ERROR", "Client should support multipart/form-data"))
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
		w.Write(Message("ERROR", "Unable to decode your image"))
		return
	}
	t0 := time.Now()

	imgBaby := resize.Resize(babyWidth, 0, img, resize.NearestNeighbor)
	imgInfant := resize.Resize(infantWidth, 0, imgBaby, resize.NearestNeighbor)
	imgNewborn := resize.Resize(newbornWidth, 0, imgInfant, resize.NearestNeighbor)
	imgSperm := resize.Resize(sperm, sperm, imgNewborn, resize.NearestNeighbor)

	red, green, blue, _ := imgSperm.At(0, 0).RGBA()
	color := fmt.Sprintf("%X%X%X", red>>8, green>>8, blue>>8) // removing 1 byte 9A16->9A

	fileOrig, err := imgToFile(img, color)
	if err != nil {
		w.Write(Message("ERROR", "Unable to save your image"))
	}
	fileBaby, err := imgToFile(imgBaby, color)
	fileInfant, err := imgToFile(imgInfant, color)
	fileNewborn, err := imgToFile(imgNewborn, color)

	result := Result{
		Egg: fileOrig,
	}

	egg := Egg{
		Egg:     fileOrig,
		Baby:    fileBaby,
		Infant:  fileInfant,
		Newborn: fileNewborn,
	}
	err = egg.saveMeta()

	if err != nil {
		w.Write(Message("ERROR", "Was not able to save your file"))
	} else {
		w.Write(Message("OK", &result))
	}

	t1 := time.Now()
	log.Printf("The call took %v to run.\n", t1.Sub(t0))
}

func genHash(img image.Image) (string, error) {
	h := sha1.New()
	err := jpeg.Encode(h, img, nil)
	return fmt.Sprintf("%x", h.Sum(nil)), err // generate hash
}

func imgToFile(img image.Image, color string) (string, error) {
	hash, err := genHash(img)
	if err != nil {
		log.Println("Unable to a file ", err)
	}
	file := genFile(hash, color, img.Bounds().Size().X, img.Bounds().Size().Y)
	path := genPath(file)
	out, err := os.Create(path)
	if err != nil {
		log.Println("Unable to create a file", err)
	}
	defer out.Close()
	err = jpeg.Encode(out, img, nil) // write image to file
	if err != nil {
		log.Println("Unable to save your image to file")
	}
	return file, err
}

func parsePath(eid string) string {
	return fmt.Sprintf(ikuraStore+"%s/%s/%s", eid[5:7], eid[7:9], eid)
}

func get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", cacheMaxAge))
	eid := html.EscapeString(r.URL.Path[5:])
	path := parsePath(eid)
	http.ServeFile(w, r, path)
}

func initStore(path string) {
	log.Println("Initializing data store...")
	for i := 0; i < 256; i++ {
		for x := 0; x < 256; x++ {
			err := os.MkdirAll(fmt.Sprintf("%s/%02x/%02x", path, i, x), 0755)
			if err != nil {
				log.Fatal("Was not able to create dirs ", err)
			}
		}
	}
	log.Println("...Done") // total 65536 directories
}

func logHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		log.Println(req.URL)
		h.ServeHTTP(rw, req)
	})
}

func main() {
	initStore(ikuraStore)
	http.HandleFunc("/", put)
	http.HandleFunc("/egg/", get)
	err := http.ListenAndServe(":9090", logHandler(http.DefaultServeMux))
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
