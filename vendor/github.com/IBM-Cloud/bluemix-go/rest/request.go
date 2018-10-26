package rest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
)

const (
	contentType               = "Content-Type"
	jsonContentType           = "application/json"
	formUrlEncodedContentType = "application/x-www-form-urlencoded"
)

// File represents a file upload in the POST request
type File struct {
	// File name
	Name string
	// File content
	Content io.Reader
	// Mime type, defaults to "application/octet-stream"
	Type string
}

// Request is a REST request. It also acts like a HTTP request builder.
type Request struct {
	method string
	rawUrl string
	header http.Header

	queryParams url.Values
	formParams  url.Values

	// files to upload
	files map[string][]File

	// custom request body
	body interface{}
}

// NewRequest creates a new REST request with the given rawUrl.
func NewRequest(rawUrl string) *Request {
	return &Request{
		rawUrl:      rawUrl,
		header:      http.Header{},
		queryParams: url.Values{},
		formParams:  url.Values{},
		files:       make(map[string][]File),
	}
}

// Method sets HTTP method of the request.
func (r *Request) Method(method string) *Request {
	r.method = method
	return r
}

// GetRequest creates a REST request with GET method and the given rawUrl.
func GetRequest(rawUrl string) *Request {
	return NewRequest(rawUrl).Method("GET")
}

// HeadRequest creates a REST request with HEAD method and the given rawUrl.
func HeadRequest(rawUrl string) *Request {
	return NewRequest(rawUrl).Method("HEAD")
}

// PostRequest creates a REST request with POST method and the given rawUrl.
func PostRequest(rawUrl string) *Request {
	return NewRequest(rawUrl).Method("POST")
}

// PutRequest creates a REST request with PUT method and the given rawUrl.
func PutRequest(rawUrl string) *Request {
	return NewRequest(rawUrl).Method("PUT")
}

// DeleteRequest creates a REST request with DELETE method and the given
// rawUrl.
func DeleteRequest(rawUrl string) *Request {
	return NewRequest(rawUrl).Method("DELETE")
}

// PatchRequest creates a REST request with PATCH method and the given
// rawUrl.
func PatchRequest(rawUrl string) *Request {
	return NewRequest(rawUrl).Method("PATCH")
}

// Creates a request with HTTP OPTIONS.
func OptionsRequest(rawUrl string) *Request {
	return NewRequest(rawUrl).Method("OPTIONS")
}

// Add adds the key, value pair to the request header. It appends to any
// existing values associated with key.
func (r *Request) Add(key string, value string) *Request {
	r.header.Add(http.CanonicalHeaderKey(key), value)
	return r
}

// Del deletes the header as specified by the key.
func (r *Request) Del(key string) *Request {
	r.header.Del(http.CanonicalHeaderKey(key))
	return r
}

// Set sets the header entries associated with key to the single element value.
// It replaces any existing values associated with key.
func (r *Request) Set(key string, value string) *Request {
	r.header.Set(http.CanonicalHeaderKey(key), value)
	return r
}

// Query appends the key, value pair to the request query which will be
// encoded as url query parameters on HTTP request's url.
func (r *Request) Query(key string, value string) *Request {
	r.queryParams.Add(key, value)
	return r
}

// Field appends the key, value pair to the form fields in the POST request.
func (r *Request) Field(key string, value string) *Request {
	r.formParams.Add(key, value)
	return r
}

// File appends a file upload item in the POST request. The file content will
// be consumed when building HTTP request (see Build()) and closed if it's
// also a ReadCloser type.
func (r *Request) File(name string, file File) *Request {
	r.files[name] = append(r.files[name], file)
	return r
}

// Body sets the request body. Accepted types are string, []byte, io.Reader,
// or structs to be JSON encodeded.
func (r *Request) Body(body interface{}) *Request {
	r.body = body
	return r
}

// Build builds a HTTP request according to the settings in the REST request.
func (r *Request) Build() (*http.Request, error) {
	url, err := r.buildURL()
	if err != nil {
		return nil, err
	}

	body, err := r.buildBody()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(r.method, url, body)
	if err != nil {
		return req, err
	}

	for k, vs := range r.header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	return req, nil
}

func (r *Request) buildURL() (string, error) {
	if r.rawUrl == "" || len(r.queryParams) == 0 {
		return r.rawUrl, nil
	}
	u, err := url.Parse(r.rawUrl)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, vs := range r.queryParams {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (r *Request) buildBody() (io.Reader, error) {
	if len(r.files) > 0 {
		return r.buildFormMultipart()
	}

	if len(r.formParams) > 0 {
		return r.buildFormFields()
	}

	return r.buildCustomBody()
}

func (r *Request) buildFormMultipart() (io.Reader, error) {
	b := new(bytes.Buffer)
	w := multipart.NewWriter(b)
	defer w.Close()

	for k, files := range r.files {
		for _, f := range files {
			defer func() {
				if f, ok := f.Content.(io.ReadCloser); ok {
					f.Close()
				}
			}()

			p, err := createPartWriter(w, k, f)
			if err != nil {
				return nil, err
			}
			_, err = io.Copy(p, f.Content)
			if err != nil {
				return nil, err
			}
		}
	}

	for k, vs := range r.formParams {
		for _, v := range vs {
			err := w.WriteField(k, v)
			if err != nil {
				return nil, err
			}
		}
	}

	r.header.Set(contentType, w.FormDataContentType())
	return b, nil
}

func createPartWriter(w *multipart.Writer, fieldName string, f File) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldName), escapeQuotes(f.Name)))
	if f.Type != "" {
		h.Set("Content-Type", f.Type)
	} else {
		h.Set("Content-Type", "application/octet-stream")
	}
	return w.CreatePart(h)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func (r *Request) buildFormFields() (io.Reader, error) {
	r.header.Set(contentType, formUrlEncodedContentType)
	return strings.NewReader(r.formParams.Encode()), nil
}

func (r *Request) buildCustomBody() (io.Reader, error) {
	if r.body == nil {
		return nil, nil
	}

	switch b := r.body; b.(type) {
	case string:
		return strings.NewReader(b.(string)), nil
	case []byte:
		return bytes.NewReader(b.([]byte)), nil
	case io.Reader:
		return b.(io.Reader), nil
	default:
		raw, err := json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("Invalid JSON request: %v", err)
		}
		return bytes.NewReader(raw), nil
	}
}
