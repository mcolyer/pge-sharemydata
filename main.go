package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
)

func main() {
	http.HandleFunc("/api/webhook/pge-daily-update", HelloServer)
	http.ListenAndServe(":8080", nil)
}

func HelloServer(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(requestDump))
}
