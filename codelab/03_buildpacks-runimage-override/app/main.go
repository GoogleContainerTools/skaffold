package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", hello)

	log.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}

func hello(w http.ResponseWriter, _ *http.Request) {
	data, err := ioutil.ReadFile("/hello.txt")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, string(data))
}
