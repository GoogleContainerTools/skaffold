package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Hexagon describes an hexagon.
type Hexagon struct {
	Image       string `json:"image"`
	Category    string `json:"category"`
	Name        string `json:"name"`
	Description string `json:"description"`
	URL         string `json:"url"`
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/api/hexagons", handleErrors(hexagons))

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", r)
}

func handleErrors(handler func(http.ResponseWriter, *http.Request) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := handler(w, r); err != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), 500)
		}
	}
}

func hexagons(w http.ResponseWriter, r *http.Request) error {
	fmt.Println("hexagons")

	db, err := sql.Open("postgres", "postgres://hexagons:notsosecret@db/hexagons?sslmode=disable")
	if err != nil {
		return err
	}

	rows, err := db.Query("SELECT image, category, name, description, url FROM hexagons")
	if err != nil {
		return err
	}

	var hexagons []Hexagon
	for rows.Next() {
		var h Hexagon
		if err := rows.Scan(&h.Image, &h.Category, &h.Name, &h.Description, &h.URL); err != nil {
			return err
		}

		hexagons = append(hexagons, h)
	}

	buf, err := json.Marshal(hexagons)
	if err != nil {
		return err
	}

	_, err = w.Write(buf)
	return err
}
