//
// Copyright 2025 The Sigstore Authors.
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

package write

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	pbs "github.com/sigstore/protobuf-specs/gen/pb-go/rekor/v1"
	"github.com/sigstore/rekor-tiles/v2/pkg/client"
	pb "github.com/sigstore/rekor-tiles/v2/pkg/generated/protobuf"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	addPath = "/api/v2/log/entries"
)

// Client writes entries to rekor.
type Client interface {
	Add(context.Context, any) (*pbs.TransparencyLogEntry, error)
}

type writeClient struct {
	baseURL *url.URL
	client  *http.Client
}

// NewWriter creates a new writer client.
func NewWriter(writeURL string, opts ...client.Option) (Client, error) {
	cfg := &client.Config{}
	for _, o := range opts {
		o(cfg)
	}
	baseURL, err := url.Parse(writeURL)
	if err != nil {
		return nil, fmt.Errorf("parsing url %s: %w", writeURL, err)
	}
	httpClient := &http.Client{
		Transport: client.CreateRoundTripper(http.DefaultTransport, cfg.UserAgent),
		Timeout:   cfg.Timeout,
	}
	return &writeClient{
		baseURL: baseURL,
		client:  httpClient,
	}, nil
}

// Add uploads a hashedrekord or DSSE log entry and returns the TransparencyLogEntry proving the entry's inclusion in the log.
func (w *writeClient) Add(ctx context.Context, entry any) (*pbs.TransparencyLogEntry, error) {
	cer, err := createRequest(entry)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	endpoint := *w.baseURL
	endpoint.Path = path.Join(endpoint.Path, addPath)

	payload, err := protojson.Marshal(cer)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewBuffer(payload))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := w.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getting response: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected response: %v %v", resp.StatusCode, string(body))
	}
	tle := pbs.TransparencyLogEntry{}
	err = protojson.Unmarshal(body, &tle)
	if err != nil {
		return nil, fmt.Errorf("unmarshaling response body: %w", err)
	}
	return &tle, nil
}

func createRequest(entry any) (*pb.CreateEntryRequest, error) {
	switch e := entry.(type) {
	case *pb.HashedRekordRequestV002:
		return createHashedRekordRequest(e), nil
	case *pb.DSSERequestV002:
		return createDSSERequest(e), nil
	default:
		return nil, fmt.Errorf("unsupported entry type: %T", entry)
	}
}

func createHashedRekordRequest(h *pb.HashedRekordRequestV002) *pb.CreateEntryRequest {
	return &pb.CreateEntryRequest{
		Spec: &pb.CreateEntryRequest_HashedRekordRequestV002{
			HashedRekordRequestV002: h,
		},
	}
}

func createDSSERequest(d *pb.DSSERequestV002) *pb.CreateEntryRequest {
	return &pb.CreateEntryRequest{
		Spec: &pb.CreateEntryRequest_DsseRequestV002{
			DsseRequestV002: d,
		},
	}
}
