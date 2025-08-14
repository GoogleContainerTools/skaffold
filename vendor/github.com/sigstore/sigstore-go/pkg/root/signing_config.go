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

package root

import (
	"fmt"
	"math/rand"
	"os"
	"slices"
	"time"

	v1 "github.com/sigstore/protobuf-specs/gen/pb-go/common/v1"
	prototrustroot "github.com/sigstore/protobuf-specs/gen/pb-go/trustroot/v1"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const SigningConfigMediaType02 = "application/vnd.dev.sigstore.signingconfig.v0.2+json"

type SigningConfig struct {
	signingConfig *prototrustroot.SigningConfig
}

type Service struct {
	URL                 string
	MajorAPIVersion     uint32
	ValidityPeriodStart time.Time
	ValidityPeriodEnd   time.Time
}

type ServiceConfiguration struct {
	Selector prototrustroot.ServiceSelector
	Count    uint32
}

// SelectService returns which service endpoint should be used based on supported API versions
// and current time. It will select the first service that matches the criteria. Services should
// be sorted from newest to oldest validity period start time, to minimize how far clients
// need to search to find a matching service.
func SelectService(services []Service, supportedAPIVersions []uint32, currentTime time.Time) (string, error) {
	for _, s := range services {
		if slices.Contains(supportedAPIVersions, s.MajorAPIVersion) && s.ValidAtTime(currentTime) {
			return s.URL, nil
		}
	}
	return "", fmt.Errorf("no matching service found for API versions %v and current time %v", supportedAPIVersions, currentTime)
}

// SelectServices returns which service endpoints should be used based on supported API versions
// and current time. It will use the configuration's selector to pick a set of services.
// ALL will return all service endpoints, ANY will return a random endpoint, and
// EXACT will return a random selection of a specified number of endpoints.
func SelectServices(services []Service, config ServiceConfiguration, supportedAPIVersions []uint32, currentTime time.Time) ([]string, error) {
	var urls []string
	for _, s := range services {
		if slices.Contains(supportedAPIVersions, s.MajorAPIVersion) && s.ValidAtTime(currentTime) {
			urls = append(urls, s.URL)
		}
	}
	if len(urls) == 0 {
		return nil, fmt.Errorf("no matching services found for API versions %v and current time %v", supportedAPIVersions, currentTime)
	}
	switch config.Selector {
	case prototrustroot.ServiceSelector_ALL:
		return urls, nil
	case prototrustroot.ServiceSelector_ANY:
		i := rand.Intn(len(urls)) // #nosec G404
		return []string{urls[i]}, nil
	case prototrustroot.ServiceSelector_EXACT:
		matchedUrls, err := selectExact(urls, config.Count)
		if err != nil {
			return nil, err
		}
		return matchedUrls, nil
	default:
		return nil, fmt.Errorf("invalid service selector")
	}
}

func selectExact[T any](slice []T, count uint32) ([]T, error) {
	if count == 0 {
		return nil, fmt.Errorf("service selector count must be greater than 0")
	}
	if int(count) > len(slice) {
		return nil, fmt.Errorf("service selector count %d must be less than or equal to the slice length %d", count, len(slice))
	}
	sliceCopy := make([]T, len(slice))
	copy(sliceCopy, slice)
	var result []T
	for range count {
		i := rand.Intn(len(sliceCopy)) // #nosec G404
		result = append(result, sliceCopy[i])
		// Remove element from slice
		sliceCopy[i], sliceCopy[len(sliceCopy)-1] = sliceCopy[len(sliceCopy)-1], sliceCopy[i]
		sliceCopy = sliceCopy[:len(sliceCopy)-1]
	}
	return result, nil
}

func mapFunc[T, V any](ts []T, fn func(T) V) []V {
	result := make([]V, len(ts))
	for i, t := range ts {
		result[i] = fn(t)
	}
	return result
}

func (s Service) ValidAtTime(t time.Time) bool {
	if !s.ValidityPeriodStart.IsZero() && t.Before(s.ValidityPeriodStart) {
		return false
	}
	if !s.ValidityPeriodEnd.IsZero() && t.After(s.ValidityPeriodEnd) {
		return false
	}
	return true
}

func (s Service) ToServiceProtobuf() *prototrustroot.Service {
	return &prototrustroot.Service{
		Url:             s.URL,
		MajorApiVersion: s.MajorAPIVersion,
		ValidFor: &v1.TimeRange{
			Start: timestamppb.New(s.ValidityPeriodStart),
			End:   timestamppb.New(s.ValidityPeriodEnd),
		},
	}
}

func (sc ServiceConfiguration) ToConfigProtobuf() *prototrustroot.ServiceConfiguration {
	return &prototrustroot.ServiceConfiguration{
		Selector: sc.Selector,
		Count:    sc.Count,
	}
}

func (sc *SigningConfig) FulcioCertificateAuthorityURLs() []Service {
	var services []Service
	for _, s := range sc.signingConfig.GetCaUrls() {
		services = append(services, Service{
			URL:                 s.GetUrl(),
			MajorAPIVersion:     s.GetMajorApiVersion(),
			ValidityPeriodStart: s.GetValidFor().GetStart().AsTime(),
			ValidityPeriodEnd:   s.GetValidFor().GetEnd().AsTime(),
		})
	}
	return services
}

func (sc *SigningConfig) OIDCProviderURLs() []Service {
	var services []Service
	for _, s := range sc.signingConfig.GetOidcUrls() {
		services = append(services, Service{
			URL:                 s.GetUrl(),
			MajorAPIVersion:     s.GetMajorApiVersion(),
			ValidityPeriodStart: s.GetValidFor().GetStart().AsTime(),
			ValidityPeriodEnd:   s.GetValidFor().GetEnd().AsTime(),
		})
	}
	return services
}

func (sc *SigningConfig) RekorLogURLs() []Service {
	var services []Service
	for _, s := range sc.signingConfig.GetRekorTlogUrls() {
		services = append(services, Service{
			URL:                 s.GetUrl(),
			MajorAPIVersion:     s.GetMajorApiVersion(),
			ValidityPeriodStart: s.GetValidFor().GetStart().AsTime(),
			ValidityPeriodEnd:   s.GetValidFor().GetEnd().AsTime(),
		})
	}
	return services
}

func (sc *SigningConfig) RekorLogURLsConfig() ServiceConfiguration {
	c := sc.signingConfig.GetRekorTlogConfig()
	return ServiceConfiguration{
		Selector: c.Selector,
		Count:    c.Count,
	}
}

func (sc *SigningConfig) TimestampAuthorityURLs() []Service {
	var services []Service
	for _, s := range sc.signingConfig.GetTsaUrls() {
		services = append(services, Service{
			URL:                 s.GetUrl(),
			MajorAPIVersion:     s.GetMajorApiVersion(),
			ValidityPeriodStart: s.GetValidFor().GetStart().AsTime(),
			ValidityPeriodEnd:   s.GetValidFor().GetEnd().AsTime(),
		})
	}
	return services
}

func (sc *SigningConfig) TimestampAuthorityURLsConfig() ServiceConfiguration {
	c := sc.signingConfig.GetTsaConfig()
	return ServiceConfiguration{
		Selector: c.Selector,
		Count:    c.Count,
	}
}

func (sc *SigningConfig) WithFulcioCertificateAuthorityURLs(fulcioURLs ...Service) *SigningConfig {
	var services []*prototrustroot.Service
	for _, u := range fulcioURLs {
		services = append(services, &prototrustroot.Service{
			Url:             u.URL,
			MajorApiVersion: u.MajorAPIVersion,
			ValidFor: &v1.TimeRange{
				Start: timestamppb.New(u.ValidityPeriodStart),
				End:   timestamppb.New(u.ValidityPeriodEnd),
			},
		})
	}
	sc.signingConfig.CaUrls = services
	return sc
}

func (sc *SigningConfig) AddFulcioCertificateAuthorityURLs(fulcioURLs ...Service) *SigningConfig {
	for _, u := range fulcioURLs {
		sc.signingConfig.CaUrls = append(sc.signingConfig.CaUrls, u.ToServiceProtobuf())
	}
	return sc
}

func (sc *SigningConfig) WithOIDCProviderURLs(oidcURLs ...Service) *SigningConfig {
	var services []*prototrustroot.Service
	for _, u := range oidcURLs {
		services = append(services, u.ToServiceProtobuf())
	}
	sc.signingConfig.OidcUrls = services
	return sc
}

func (sc *SigningConfig) AddOIDCProviderURLs(oidcURLs ...Service) *SigningConfig {
	for _, u := range oidcURLs {
		sc.signingConfig.OidcUrls = append(sc.signingConfig.OidcUrls, u.ToServiceProtobuf())
	}
	return sc
}

func (sc *SigningConfig) WithRekorLogURLs(logURLs ...Service) *SigningConfig {
	var services []*prototrustroot.Service
	for _, u := range logURLs {
		services = append(services, u.ToServiceProtobuf())
	}
	sc.signingConfig.RekorTlogUrls = services
	return sc
}

func (sc *SigningConfig) AddRekorLogURLs(logURLs ...Service) *SigningConfig {
	for _, u := range logURLs {
		sc.signingConfig.RekorTlogUrls = append(sc.signingConfig.RekorTlogUrls, u.ToServiceProtobuf())
	}
	return sc
}

func (sc *SigningConfig) WithRekorTlogConfig(selector prototrustroot.ServiceSelector, count uint32) *SigningConfig {
	sc.signingConfig.RekorTlogConfig.Selector = selector
	sc.signingConfig.RekorTlogConfig.Count = count
	return sc
}

func (sc *SigningConfig) WithTimestampAuthorityURLs(tsaURLs ...Service) *SigningConfig {
	var services []*prototrustroot.Service
	for _, u := range tsaURLs {
		services = append(services, u.ToServiceProtobuf())
	}
	sc.signingConfig.TsaUrls = services
	return sc
}

func (sc *SigningConfig) AddTimestampAuthorityURLs(tsaURLs ...Service) *SigningConfig {
	for _, u := range tsaURLs {
		sc.signingConfig.TsaUrls = append(sc.signingConfig.TsaUrls, u.ToServiceProtobuf())
	}
	return sc
}

func (sc *SigningConfig) WithTsaConfig(selector prototrustroot.ServiceSelector, count uint32) *SigningConfig {
	sc.signingConfig.TsaConfig.Selector = selector
	sc.signingConfig.TsaConfig.Count = count
	return sc
}

func (sc SigningConfig) String() string {
	return fmt.Sprintf("{CA: %v, OIDC: %v, RekorLogs: %v, TSAs: %v, MediaType: %s}",
		sc.FulcioCertificateAuthorityURLs(),
		sc.OIDCProviderURLs(),
		sc.RekorLogURLs(),
		sc.TimestampAuthorityURLs(),
		SigningConfigMediaType02)
}

// NewSigningConfig initializes a SigningConfig object from a mediaType string, Fulcio certificate
// authority URLs, OIDC provider URLs, Rekor transparency log URLs, timestamp authorities URLs,
// selection criteria for Rekor logs and TSAs.
func NewSigningConfig(mediaType string,
	fulcioCertificateAuthorities []Service,
	oidcProviders []Service,
	rekorLogs []Service,
	rekorLogsConfig ServiceConfiguration,
	timestampAuthorities []Service,
	timestampAuthoritiesConfig ServiceConfiguration) (*SigningConfig, error) {
	if mediaType != SigningConfigMediaType02 {
		return nil, fmt.Errorf("unsupported SigningConfig media type, must be: %s", SigningConfigMediaType02)
	}
	sc := &SigningConfig{
		signingConfig: &prototrustroot.SigningConfig{
			MediaType:       mediaType,
			CaUrls:          mapFunc(fulcioCertificateAuthorities, Service.ToServiceProtobuf),
			OidcUrls:        mapFunc(oidcProviders, Service.ToServiceProtobuf),
			RekorTlogUrls:   mapFunc(rekorLogs, Service.ToServiceProtobuf),
			RekorTlogConfig: rekorLogsConfig.ToConfigProtobuf(),
			TsaUrls:         mapFunc(timestampAuthorities, Service.ToServiceProtobuf),
			TsaConfig:       timestampAuthoritiesConfig.ToConfigProtobuf(),
		},
	}
	return sc, nil
}

// NewSigningConfigFromProtobuf returns a Sigstore signing configuration.
func NewSigningConfigFromProtobuf(sc *prototrustroot.SigningConfig) (*SigningConfig, error) {
	if sc.GetMediaType() != SigningConfigMediaType02 {
		return nil, fmt.Errorf("unsupported SigningConfig media type: %s", sc.GetMediaType())
	}
	return &SigningConfig{signingConfig: sc}, nil
}

// NewSigningConfigFromPath returns a Sigstore signing configuration from a file.
func NewSigningConfigFromPath(path string) (*SigningConfig, error) {
	scJSON, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return NewSigningConfigFromJSON(scJSON)
}

// NewSigningConfigFromJSON returns a Sigstore signing configuration from JSON.
func NewSigningConfigFromJSON(rootJSON []byte) (*SigningConfig, error) {
	pbSC, err := NewSigningConfigProtobuf(rootJSON)
	if err != nil {
		return nil, err
	}

	return NewSigningConfigFromProtobuf(pbSC)
}

// NewSigningConfigProtobuf returns a Sigstore signing configuration as a protobuf.
func NewSigningConfigProtobuf(scJSON []byte) (*prototrustroot.SigningConfig, error) {
	pbSC := &prototrustroot.SigningConfig{}
	err := protojson.Unmarshal(scJSON, pbSC)
	if err != nil {
		return nil, err
	}
	return pbSC, nil
}

// FetchSigningConfig fetches the public-good Sigstore signing configuration from TUF.
func FetchSigningConfig() (*SigningConfig, error) {
	return FetchSigningConfigWithOptions(tuf.DefaultOptions())
}

// FetchSigningConfig fetches the public-good Sigstore signing configuration with the given options from TUF.
func FetchSigningConfigWithOptions(opts *tuf.Options) (*SigningConfig, error) {
	client, err := tuf.New(opts)
	if err != nil {
		return nil, err
	}
	return GetSigningConfig(client)
}

// FetchSigningConfig fetches the public-good Sigstore signing configuration target from TUF.
func GetSigningConfig(c *tuf.Client) (*SigningConfig, error) {
	jsonBytes, err := c.GetTarget("signing_config.json")
	if err != nil {
		return nil, err
	}
	return NewSigningConfigFromJSON(jsonBytes)
}
