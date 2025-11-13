package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	message := os.Getenv("MESSAGE")
	if message == "" {
		message = "Hello, World!"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s\n", message)
	})

	port := ":8080"
	log.Printf("Server starting on port %s", port)
	log.Printf("Message: %s", message)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
