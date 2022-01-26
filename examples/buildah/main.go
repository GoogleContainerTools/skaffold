package main

import (
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", helloWorldHandler)

	log.Println("starting http handler on :8080")
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func helloWorldHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello-world!"))
}
