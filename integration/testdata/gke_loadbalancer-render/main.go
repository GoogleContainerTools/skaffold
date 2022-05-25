package main

import (
	"fmt"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello!!\n")
}

func main() {
	log.Print("gke loadbalancer server ready")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":3000", nil)
}
