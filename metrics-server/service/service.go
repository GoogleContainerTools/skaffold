/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package service

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"cloud.google.com/go/storage"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	log.Print("skaffold metrics server received a request")
	ctx := context.Background()

	// Creates a client.
	client, err := storage.NewClient(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		log.Printf("Failed to create client: %v", err)
	}

	// Sets the name for the new bucket.
	bucketName := os.Getenv("SKAFFOLD_SURVEY_BUCKET")
	if bucketName == "" {
		bucketName = "test"
	}
	w.Write([]byte(bucketName))
	//body, err := r.GetBody()
	//if err != nil {
	//	w.WriteHeader(http.StatusInternalServerError)
	//	w.Write([]byte(err.Error()))
	//	return
	//}

	sr := strings.NewReader("received ping")
	wc := client.Bucket(bucketName).Object("metrics-test").NewWriter(ctx)
	if _, err = io.Copy(wc, sr); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err.Error())
		w.Write([]byte(err.Error()))
	}

	if err := wc.Close(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
	}
}