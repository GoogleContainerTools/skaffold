// Package rest provides a simple REST client for creating and sending
// API requests.

// Examples:
// Creating request
// 		// GET request
// 		GetRequest("http://www.example.com").
// 			Set("Accept", "application/json").
// 			Query("foo1", "bar1").
// 			Query("foo2", "bar2")
//
// 		// JSON body
// 		foo = Foo{Bar: "val"}
// 		PostRequest("http://www.example.com").
// 			Body(foo)

// 		// String body
// 		PostRequest("http://www.example.com").
// 			Body("{\"bar\": \"val\"}")

// 		// Stream body
// 		PostRequest("http://www.example.com").
// 			Body(strings.NewReader("abcde"))

// 		// Multipart POST request
// 		var f *os.File
// 		PostRequest("http://www.example.com").
// 			Field("foo", "bar").
// 			File("file1", File{Name: f.Name(), Content: f}).
// 			File("file2", File{Name: "1.txt", Content: []byte("abcde"), Type: "text/plain")

// 		// Build to an HTTP request
// 		GetRequest("http://www.example.com").Build()

// Sending request:
// 		client := NewClient()
// 		var foo = struct {
// 			Bar string
// 		}{}
// 		var apiErr = struct {
// 			Message string
// 		}{}
// 		resp, err := client.Do(request, &foo, &apiErr)
package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/IBM-Cloud/bluemix-go/bmxerror"
)

const (
	//ErrCodeEmptyResponse ...
	ErrCodeEmptyResponse = "EmptyResponseBody"
)

//ErrEmptyResponseBody ...
var ErrEmptyResponseBody = bmxerror.New(ErrCodeEmptyResponse, "empty response body")

// Client is a REST client. It's recommend that a client be created with the
// NewClient() method.
type Client struct {
	// The HTTP client to be used. Default is HTTP's defaultClient.
	HTTPClient *http.Client
	// Defaualt header for all outgoing HTTP requests.
	DefaultHeader http.Header
}

// NewClient creates a new REST client.
func NewClient() *Client {
	return &Client{
		HTTPClient: http.DefaultClient,
	}
}

// Do sends an request and returns an HTTP response. The resp.Body will be
// consumed and closed in the method.
//
// For 2XX response, it will be JSON decoded into the value pointed to by
// respv.
//
// For non-2XX response, an attempt will be made to unmarshal the response
// into the value pointed to by errV. If unmarshal failed, an ErrorResponse
// error with status code and response text is returned.
func (c *Client) Do(r *Request, respV interface{}, errV interface{}) (*http.Response, error) {
	req, err := c.makeRequest(r)
	if err != nil {
		return nil, err
	}

	client := c.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		raw, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return resp, fmt.Errorf("Error reading response: %v", err)
		}

		if len(raw) > 0 && errV != nil {
			if json.Unmarshal(raw, errV) == nil {
				return resp, nil
			}
		}

		return resp, bmxerror.NewRequestFailure("ServerErrorResponse", string(raw), resp.StatusCode)
	}

	if respV != nil {
		// Callback function with execpted JSON type
		if funcType := reflect.TypeOf(respV); funcType.Kind() == reflect.Func {
			if funcType.NumIn() != 1 || funcType.NumOut() != 1 {
				err = fmt.Errorf("Callback funcion not expected signature: func(interface{}) bool")
			}
			paramType := funcType.In(0)
			dc := json.NewDecoder(resp.Body)
			dc.UseNumber()
			for {
				typedInterface := reflect.New(paramType).Interface()
				if err = dc.Decode(typedInterface); err == io.EOF {
					err = nil
					break
				} else if err != nil {
					break
				}
				resv := reflect.ValueOf(respV).Call([]reflect.Value{reflect.ValueOf(typedInterface).Elem()})[0]
				if !resv.Bool() {
					break
				}
			}
		} else {
			switch respV.(type) {
			case io.Writer:
				_, err = io.Copy(respV.(io.Writer), resp.Body)
			default:
				dc := json.NewDecoder(resp.Body)
				dc.UseNumber()
				err = dc.Decode(respV)
				if err == io.EOF {
					err = ErrEmptyResponseBody
				}
			}
		}
	}

	return resp, err
}

func (c *Client) makeRequest(r *Request) (*http.Request, error) {
	req, err := r.Build()
	if err != nil {
		return nil, err
	}

	c.applyDefaultHeader(req)

	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("Accept-Language") == "" {
		req.Header.Set("Accept-Language", "en")
	}

	return req, nil
}

func (c *Client) applyDefaultHeader(req *http.Request) {
	for k, vs := range c.DefaultHeader {
		if req.Header.Get(k) != "" {
			continue
		}
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
}
