package main

import (
	"fmt"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello %s", os.Getenv("GREET"))
	})

	http.ListenAndServe(":80", nil)
}
