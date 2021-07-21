package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	// adding a comment for dev loop to trigger
	fmt.Fprintf(w, "leeroooooy app!!\n")
}

func main() {
	log.Print("leeroy app server ready")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":50051", nil)
}
