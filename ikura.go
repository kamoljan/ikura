package ikura

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
	"labix.org/v2/mgo/bson"

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
	Newborn string `json:"newborn"` //0001_040db0bc2fc49ab41fd81294c7d195c7d1de358b_ACA0AC_100_160
}

type Msg struct {
	Status string      `json:"status"` //"ok"
	Result interface{} `json:"result"` //{newborn: "0001_040db0bc2fc49ab41fd81294c7d195c7d1de358b_ACA0AC_100_160"}
}

func (egg *Egg) saveMeta() error {
	session, err := mgo.Dial(mongodb)
	if err != nil {
		log.Fatal("Unable to connect to DB ", err)
	}

	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("sa").C("egg")
	// i := bson.NewObjectId() // in case we want to know _id
	// err = c.Insert(bson.M{"_id": i, "egg": &egg.Egg, "baby": &egg.Baby, "infant": &egg.Infant, "newborn": &egg.Newborn})
	err = c.Insert(&egg)
	if err != nil {
		log.Fatal("Unable to save to DB ", err)
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
		log.Println("Unable to json.Marshal ", err)
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
 * 	result: { newborn: "0001_040db0bc2fc49ab41fd81294c7d195c7d1de358b_ACA0AC_100_160" }
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
		Newborn: fileNewborn,
	}

	egg := Egg{
		Egg:     fileOrig,
		Baby:    fileBaby,
		Infant:  fileInfant,
		Newborn: fileNewborn,
	}
	err = egg.saveMeta()

	if err != nil {
		w.Write(Message("ERROR", "Unable to save your image"))
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
		log.Println("Unable to save a file ", err)
		return "", err
	}
	file := genFile(hash, color, img.Bounds().Size().X, img.Bounds().Size().Y)
	path := genPath(file)
	out, err := os.Create(path)
	if err != nil {
		log.Println("Unable to create a file", err)
		return "", err
	}
	defer out.Close()
	err = jpeg.Encode(out, img, nil) // write image to file
	if err != nil {
		log.Println("Unable to save your image to file")
		return "", err
	}
	return file, err
}

func parsePath(eid string) string {
	return fmt.Sprintf(ikuraStore+"%s/%s/%s", eid[5:7], eid[7:9], eid)
}

func getEggBySize(size, id string) (Egg, error) {
	session, err := mgo.Dial("mongodb://admin:12345678@localhost:27017/sa")
	if err != nil {
		log.Fatal("Unable to connect to DB ", err)
	}

	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	result := Egg{}
	c := session.DB("sa").C("egg")
	// err = c.FindId(bson.ObjectIdHex(id)).One(&result)
	err = c.Find(bson.M{size: id}).One(&result)
	return result, err
}

//http://localhost:9090/egg/0001_8787bec619ff019fd17fe02599a384d580bf6779_9BA4AA_400_300?type=baby
func get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%d", cacheMaxAge))
	// log.Println("GET: r.URL.Path = " + r.URL.Path)
	// log.Println("GET: r.FromValue(size) = " + r.FormValue("size"))
	size := r.FormValue("size")
	eid := html.EscapeString(r.URL.Path[5:]) //cutting "/egg/"
	if size != "" {
		d, err := getEggBySize(size, eid)
		if err != nil {
			w.Write(Message("ERROR", "Unable to find by size"))
			return
		}
		if size == "baby" {
			eid = d.Baby
		} else if size == "infant" {
			eid = d.Infant
		} else if size == "newborn" {
			eid = d.Newborn
		} else {
			eid = d.Egg
		}
	}
	log.Println("GET: eid = " + eid)
	path := parsePath(eid)
	log.Println("GET: path = " + path)
	http.ServeFile(w, r, path)
}
