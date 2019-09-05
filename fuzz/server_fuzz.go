// +build gofuzz

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

package fuzz

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/config"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/server"
	_ "github.com/dvyukov/go-fuzz/go-fuzz-dep"
)

// FuzzServerTCP fuzzes binary requests to the control API.
func FuzzServerTCP(fuzz []byte) int {
	// Ignore empty fuzz
	if len(fuzz) == 0 {
		return -1
	}

	// Start control API server
	shutdown, err := server.Initialize(config.SkaffoldOptions{})
	defer shutdown()
	if err != nil {
		panic(err)
	}
	port := server.GetHTTPPort()

	// Connect to control API
	hostport := fmt.Sprintf("localhost:%d", port)
	connection, err := net.Dial("tcp", hostport)
	if err != nil {
		panic(err)
	}

	// Deliver fuzz
	_, err = connection.Write(fuzz)
	if err != nil {
		return 0
	}
	_, err = ioutil.ReadAll(connection)
	if err != nil {
		return 0
	}
	err = connection.Close()
	if err != nil {
		return 0
	}
	return 1
}

// FuzzServerHTTP fuzzes HTTP requests to the control API.
func FuzzServerHTTP(fuzz []byte) int {
	// Decode request
	method, path, headers, body, ok := decodeFuzzRequest(fuzz)
	if !ok {
		return -1
	}

	// Start control API server
	shutdown, err := server.Initialize(config.SkaffoldOptions{})
	defer shutdown()
	if err != nil {
		panic(err)
	}
	port := server.GetHTTPPort()

	// Deliver fuzz
	client := &http.Client{}
	address := fmt.Sprintf("http://localhost:%s/%s", port, path)
	request, err := http.NewRequest(method, address, bytes.NewReader(body))
	if err != nil {
		return -1
	}
	for name, values := range headers {
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	_, err = client.Do(request)
	if err != nil {
		return 0
	}
	return 1
}

func decodeFuzzRequest(fuzz []byte) (
	method string,
	path string,
	headers map[string][]string,
	body []byte,
	ok bool,
) {
	if len(fuzz) < 3 {
		return
	}
	method = decodeFuzzMethod(fuzz)
	if method == "" {
		return
	}
	fuzz = fuzz[1:]
	path, fuzz, ok = extractFuzzString(fuzz)
	if !ok {
		return
	}
	headers = make(map[string][]string)
	fuzz, ok = decodeFuzzHeaders(fuzz, headers)
	if !ok {
		return
	}
	body = fuzz
	ok = true
	return
}

// https://www.iana.org/assignments/http-methods/http-methods.xhtml
func decodeFuzzMethod(fuzz []byte) (method string) {
	switch fuzz[0] {
	case 0:
		return "ACL"
	case 1:
		return "BASELINE-CONTROL"
	case 2:
		return "BIND"
	case 3:
		return "CHECKIN"
	case 4:
		return "CHECKOUT"
	case 5:
		return "CONNECT"
	case 6:
		return "COPY"
	case 7:
		return "DELETE"
	case 8:
		return "GET"
	case 9:
		return "HEAD"
	case 10:
		return "LABEL"
	case 11:
		return "LINK"
	case 12:
		return "LOCK"
	case 13:
		return "MERGE"
	case 14:
		return "MKACTIVITY"
	case 15:
		return "MKCALENDAR"
	case 16:
		return "MKCOL"
	case 17:
		return "MKREDIRECTREF"
	case 18:
		return "MKWORKSPACE"
	case 19:
		return "MOVE"
	case 20:
		return "OPTIONS"
	case 21:
		return "ORDERPATCH"
	case 22:
		return "PATCH"
	case 23:
		return "POST"
	case 24:
		return "PRI"
	case 25:
		return "PROPFIND"
	case 26:
		return "PROPPATCH"
	case 27:
		return "PUT"
	case 28:
		return "REBIND"
	case 29:
		return "REPORT"
	case 30:
		return "SEARCH"
	case 31:
		return "TRACE"
	case 32:
		return "UNBIND"
	case 33:
		return "UNCHECKOUT"
	case 34:
		return "UNLINK"
	case 35:
		return "UNLOCK"
	case 36:
		return "UPDATE"
	case 37:
		return "UPDATEREDIRECTREF"
	case 38:
		return "VERSION-CONTROL"
	default:
		return ""
	}
}

func decodeFuzzHeaders(fuzz []byte, headers map[string][]string) (
	rest []byte,
	ok bool,
) {
	rest = fuzz
	for {
		if len(rest) == 0 {
			// Consumed all fuzz
			ok = true
			return
		}
		if fuzz[0] == 0 {
			// Headers terminated
			if len(rest) == 1 {
				rest = []byte{}
			} else {
				rest = rest[1:]
			}
			ok = true
			return
		}
		if len(fuzz) == 1 {
			// Invalid headers encoding
			return
		}
		rest, ok = decodeFuzzHeader(rest[1:], headers)
		if !ok {
			return
		}
	}
}

func decodeFuzzHeader(fuzz []byte, headers map[string][]string) (
	rest []byte,
	ok bool,
) {
	if len(fuzz) == 0 {
		ok = true
		return
	}
	name, rest, ok := extractFuzzString(fuzz)
	if !ok {
		return
	}
	value, rest, ok := extractFuzzString(rest)
	if !ok {
		return
	}
	if header, ok := headers[name]; ok {
		headers[name] = append(header, value)
	} else {
		headers[name] = []string{value}
	}
	ok = true
	return
}

func extractFuzzString(fuzz []byte) (
	value string,
	rest []byte,
	ok bool,
) {
	if len(fuzz) < 2 {
		// Invalid string encoding
		return
	}
	length := int(fuzz[0])
	if length == 0 {
		// Invalid length
		return
	}
	if len(fuzz) < (length + 1) {
		// Insufficient fuzz
		return
	}
	value = string(fuzz[1 : length+1])
	if len(fuzz) == (length + 1) {
		// Consumed all fuzz
		rest = []byte{}
	} else {
		// More fuzz
		rest = fuzz[length+1:]
	}
	ok = true
	return
}
