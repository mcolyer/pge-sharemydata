package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
)

type BatchList struct {
	Resources []Resource `xml:"resources"`
}
type Resource struct {
	Value string `xml:",innerxml"`
}

func main() {
	http.HandleFunc("/api/webhook/pge-daily-update", HelloServer)
	http.ListenAndServe(":8080", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)

	resources := ParseRequestBody(r.Body)

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
}

func ParseRequestBody(reader io.Reader) []Resource {
	var data BatchList
	s, err := ioutil.ReadAll(reader)
	if err != nil {
		log.Fatal(err)
	}

	xml.Unmarshal([]byte(s), &data)
	fmt.Printf("%v\n", data.Resources[1].Value)
	return data.Resources
}
