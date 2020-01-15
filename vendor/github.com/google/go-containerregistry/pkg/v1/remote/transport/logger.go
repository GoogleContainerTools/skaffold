package transport

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/google/go-containerregistry/pkg/logs"
)

type logTransport struct {
	inner http.RoundTripper
}

// NewLogger returns a transport that logs requests and responses to
// github.com/google/go-containerregistry/pkg/logs.Debug.
func NewLogger(inner http.RoundTripper) http.RoundTripper {
	return &logTransport{inner}
}

func (t *logTransport) RoundTrip(in *http.Request) (out *http.Response, err error) {
	// Inspired by: github.com/motemen/go-loghttp
	logs.Debug.Printf("--> %s %s", in.Method, in.URL)
	b, err := httputil.DumpRequestOut(in, true)
	if err == nil {
		logs.Debug.Println(string(b))
	}
	out, err = t.inner.RoundTrip(in)
	if err != nil {
		logs.Debug.Printf("<-- %v %s", err, in.URL)
	}
	if out != nil {
		msg := fmt.Sprintf("<-- %d", out.StatusCode)
		if out.Request != nil {
			msg = fmt.Sprintf("%s %s", msg, out.Request.URL)
		}
		logs.Debug.Print(msg)
		b, err := httputil.DumpResponse(out, true)
		if err == nil {
			logs.Debug.Println(string(b))
		}
	}
	return
}
