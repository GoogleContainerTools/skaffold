/*
Copyright 2018 The Skaffold Authors

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

package main

import (
	"log"
	"net/http"
)

const (
	port = ":8080"
)

func main() {
	//Setup the serve route to receive guthub events
	http.HandleFunc("/receive", handleGithubEvent)

	// Start the server
	log.Println("Listening...")
	log.Fatal(http.ListenAndServe(port, nil))
}

func handleGithubEvent(w http.ResponseWriter, r *http.Request) {
	// TODO (priyawadhwa@): Add logic to handle a Github event here
}
