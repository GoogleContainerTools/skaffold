package main

import (
	"fmt"
	"html"
	"log"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}

func main() {
	log.Print("Go web app ready on port 8080")
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
