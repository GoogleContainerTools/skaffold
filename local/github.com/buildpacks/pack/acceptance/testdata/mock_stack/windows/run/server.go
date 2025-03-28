/*
-p="8080": port to expose
-g:        file globs to read (comma-separated)
*/
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	port := flag.String("p", "8080", "port to expose")
	glob := flag.String("g", "", "file globs to read")
	flag.Parse()

	var resp []string

	globs := strings.Split(*glob, ",")
	for _, glob := range globs {
		paths, err := filepath.Glob(strings.TrimSpace(glob))
		if err != nil {
			panic(err.Error())
		}

		for _, path := range paths {
			contents, err := os.ReadFile(filepath.Clean(path))
			if err != nil {
				panic(err.Error())
			}

			resp = append(resp, string(contents))
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(strings.Join(resp, "\n")))
	})

	log.Printf("Serving %s on HTTP port: %s\n", *glob, *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
