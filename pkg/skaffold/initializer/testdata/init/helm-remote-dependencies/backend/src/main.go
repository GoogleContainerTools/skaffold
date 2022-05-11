package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// guestbookEntry represents the message object returned in the API.
type guestbookEntry struct {
	Author  string    `json:"author" bson:"author"`
	Message string    `json:"message" bson:"message"`
	Date    time.Time `json:"date" bson:"date"`
}

type guestbookServer struct {
	db database
}

// main starts a server listening on $PORT responding to requests "GET
// /messages" and "POST /messages" with a JSON API.
func main() {
	ctx := context.Background()

	// PORT environment variable is set in guestbook-backend.deployment.yaml.
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("PORT environment variable not specified")
	}
	// GUESTBOOK_DB_ADDR environment variable is set in guestbook-backend.deployment.yaml.
	dbAddr := os.Getenv("GUESTBOOK_DB_ADDR")
	if dbAddr == "" {
		log.Fatal("GUESTBOOK_DB_ADDR environment variable not specified")
	}

	mongoURI := "mongodb://" + dbAddr
	connCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()
	dbConn, err := mongo.Connect(connCtx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("failed to initialize connection to mongodb: %+v", err)
	}
	if err := dbConn.Ping(connCtx, readpref.Primary()); err != nil {
		log.Fatalf("ping to mongodb failed: %+v", err)
	}

	gs := &guestbookServer{
		db: &mongodb{
			conn: dbConn,
		},
	}

	log.Printf("backend server listening on port %s", port)
	http.Handle("/messages", gs)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func (s *guestbookServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("received request: method=%s path=%s", r.Method, r.URL.Path)
	if r.Method == http.MethodGet {
		s.getMessagesHandler(w, r)
	} else if r.Method == http.MethodPost {
		s.postMessageHandler(w, r)
	} else {
		http.Error(w, fmt.Sprintf("unsupported method %s", r.Method), http.StatusMethodNotAllowed)
	}
}

func (s *guestbookServer) getMessagesHandler(w http.ResponseWriter, r *http.Request) {
	entries, err := s.db.entries(r.Context())
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read entries: %+v", err), http.StatusInternalServerError)
		// TODO return JSON error
		return
	}
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		log.Printf("WARNING: failed to encode json into response: %+v", err)
	} else {
		log.Printf("%d entries returned", len(entries))
	}
}

func (s *guestbookServer) postMessageHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var v guestbookEntry
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode request body into json: %+v", err), http.StatusBadRequest)
		return
	}
	if v.Author == "" {
		http.Error(w, "empty 'author' value", http.StatusBadRequest)
		return
	}
	if v.Message == "" {
		http.Error(w, "empty 'message' value", http.StatusBadRequest)
		return
	}

	v.Date = time.Now()

	if err := s.db.addEntry(r.Context(), v); err != nil {
		http.Error(w, fmt.Sprintf("failed to save entry: %+v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("entry saved: author=%q message=%q", v.Author, v.Message)
}
