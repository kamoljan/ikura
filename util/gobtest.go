//Encoding a map to a gob.  Save the gob to disk. Read the gob from disk. Decode the gob into another map.
package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"io/ioutil"
)

func main() {
	//map literal to initialize a map with data
	thing := map[int]string{1: "one", 2: "two", 3: "three"}
	//initialize a *bytes.Buffer
	m := new(bytes.Buffer)
	//the *bytes.Buffer satisfies the io.Writer interface and can
	//be used in gob.NewEncoder()
	enc := gob.NewEncoder(m)
	//gob.Encoder has method Encode that accepts data items as parameter
	enc.Encode(thing)
	//the bytes.Buffer type has method Bytes() that returns type []byte,
	//and can be used as a parameter in ioutil.WriteFile()
	err := ioutil.WriteFile("gob", m.Bytes(), 0600)
	if err != nil {
		panic(err)
	}
	fmt.Printf("just saved gob with %v\n", thing)
	//read the file that was just written, n is []byte
	n, err := ioutil.ReadFile("gob")
	if err != nil {
		panic(err)
	}
	//create a bytes.Buffer type with n, type []byte
	p := bytes.NewBuffer(n)
	//bytes.Buffer satisfies the interface for io.Writer and can be used
	//in gob.NewDecoder()
	dec := gob.NewDecoder(p)
	//make a map reference type that we'll populate with the decoded gob
	e := make(map[int]string)
	//we must decode into a pointer, so we'll take the address of e
	err = dec.Decode(&e)
	if err != nil {
		panic(err)
	}
	fmt.Printf("just read gob from file and it's showing: %v\n", e)
}
