package main

import (
	"fmt"
	"log"
        "os"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "leeroooooy app!!\n")
}

func main() {
	log.Print("leeroy app server ready")
	os.Exit(1)
        http.HandleFunc("/", handler)
	http.ListenAndServe(":50051", nil)
}
