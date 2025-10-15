// Copyright 2024 The Sigstore Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sign

import (
	"bytes"
	"context"
	"crypto"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/digitorus/timestamp"

	"github.com/sigstore/sigstore-go/pkg/util"
)

type TimestampAuthorityOptions struct {
	// Full URL (with path) of Timestamp Authority endpoint
	URL string
	// Optional timeout for network requests (default 30s; use negative value for no timeout)
	Timeout time.Duration
	// Optional number of times to retry on HTTP 5XX
	Retries uint
	// Optional Transport (for dependency injection)
	Transport http.RoundTripper
}

type TimestampAuthority struct {
	options *TimestampAuthorityOptions
	client  *http.Client
}

var TimestampAuthorityAPIVersions = []uint32{1}

func NewTimestampAuthority(opts *TimestampAuthorityOptions) *TimestampAuthority {
	ta := &TimestampAuthority{options: opts}
	ta.client = &http.Client{
		Transport: opts.Transport,
	}

	if opts.Timeout >= 0 {
		if opts.Timeout == 0 {
			opts.Timeout = 30 * time.Second
		}
		ta.client.Timeout = opts.Timeout
	}

	return ta
}

func (ta *TimestampAuthority) GetTimestamp(ctx context.Context, signature []byte) ([]byte, error) {
	signatureHash := sha256.Sum256(signature)

	req := &timestamp.Request{
		HashAlgorithm: crypto.SHA256,
		HashedMessage: signatureHash[:],
	}
	reqBytes, err := req.Marshal()
	if err != nil {
		return nil, err
	}

	attempts := uint(0)
	var response *http.Response

	for attempts <= ta.options.Retries {
		request, err := http.NewRequest("POST", ta.options.URL, bytes.NewReader(reqBytes))
		if err != nil {
			return nil, err
		}
		request.Header.Add("Content-Type", "application/timestamp-query")
		request.Header.Add("User-Agent", util.ConstructUserAgent())

		response, err = ta.client.Do(request)
		if err != nil {
			return nil, err
		}

		if (response.StatusCode < 500 || response.StatusCode >= 600) && response.StatusCode != 429 {
			// Not a retryable HTTP status code, so don't retry
			break
		}

		delay := time.Duration(math.Pow(2, float64(attempts)))
		timer := time.NewTimer(delay * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
		attempts++
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode != 200 && response.StatusCode != 201 {
		return nil, fmt.Errorf("timestamp authority returned %d: %s", response.StatusCode, string(body))
	}

	_, err = timestamp.ParseResponse(body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
