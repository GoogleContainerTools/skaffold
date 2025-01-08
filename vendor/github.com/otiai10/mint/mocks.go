package mint

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

type HTTPClientMock struct {
	HTTPError          error
	ResponseStatusCode int
	ResponseBody       string
}

func (hcm *HTTPClientMock) Handle() (res *http.Response, err error, ok bool) {
	if hcm.HTTPError != nil {
		err = hcm.HTTPError
		ok = true
	}
	res = new(http.Response)
	if hcm.ResponseBody != "" {
		res.Body = ioutil.NopCloser(bytes.NewBufferString(hcm.ResponseBody))
		ok = true
	}
	if hcm.ResponseStatusCode != 0 {
		res.StatusCode = hcm.ResponseStatusCode
		ok = true
	}
	return res, err, ok
}
