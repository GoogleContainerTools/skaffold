package mocks

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/jmhodges/clock"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	jose "gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	berrors "github.com/letsencrypt/boulder/errors"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/mail"
	"github.com/letsencrypt/boulder/probs"
	pubpb "github.com/letsencrypt/boulder/publisher/proto"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

// StorageAuthorityReadOnly is a mock of sapb.StorageAuthorityReadOnlyClient
type StorageAuthorityReadOnly struct {
	clk clock.Clock
}

// NewStorageAuthorityReadOnly creates a new mock read-only storage authority
// with the given clock.
func NewStorageAuthorityReadOnly(clk clock.Clock) *StorageAuthorityReadOnly {
	return &StorageAuthorityReadOnly{clk}
}

// StorageAuthority is a mock of sapb.StorageAuthorityClient
type StorageAuthority struct {
	StorageAuthorityReadOnly
}

// NewStorageAuthority creates a new mock storage authority
// with the given clock.
func NewStorageAuthority(clk clock.Clock) *StorageAuthority {
	return &StorageAuthority{StorageAuthorityReadOnly{clk}}
}

const (
	test1KeyPublicJSON  = `{"kty":"RSA","n":"yNWVhtYEKJR21y9xsHV-PD_bYwbXSeNuFal46xYxVfRL5mqha7vttvjB_vc7Xg2RvgCxHPCqoxgMPTzHrZT75LjCwIW2K_klBYN8oYvTwwmeSkAz6ut7ZxPv-nZaT5TJhGk0NT2kh_zSpdriEJ_3vW-mqxYbbBmpvHqsa1_zx9fSuHYctAZJWzxzUZXykbWMWQZpEiE0J4ajj51fInEzVn7VxV-mzfMyboQjujPh7aNJxAWSq4oQEJJDgWwSh9leyoJoPpONHxh5nEE5AjE01FkGICSxjpZsF-w8hOTI3XXohUdu29Se26k2B0PolDSuj0GIQU6-W9TdLXSjBb2SpQ","e":"AQAB"}`
	test2KeyPublicJSON  = `{"kty":"RSA","n":"qnARLrT7Xz4gRcKyLdydmCr-ey9OuPImX4X40thk3on26FkMznR3fRjs66eLK7mmPcBZ6uOJseURU6wAaZNmemoYx1dMvqvWWIyiQleHSD7Q8vBrhR6uIoO4jAzJZR-ChzZuSDt7iHN-3xUVspu5XGwXU_MVJZshTwp4TaFx5elHIT_ObnTvTOU3Xhish07AbgZKmWsVbXh5s-CrIicU4OexJPgunWZ_YJJueOKmTvnLlTV4MzKR2oZlBKZ27S0-SfdV_QDx_ydle5oMAyKVtlAV35cyPMIsYNwgUGBCdY_2Uzi5eX0lTc7MPRwz6qR1kip-i59VcGcUQgqHV6Fyqw","e":"AQAB"}`
	testE1KeyPublicJSON = `{"kty":"EC","crv":"P-256","x":"FwvSZpu06i3frSk_mz9HcD9nETn4wf3mQ-zDtG21Gao","y":"S8rR-0dWa8nAcw1fbunF_ajS3PQZ-QwLps-2adgLgPk"}`
	testE2KeyPublicJSON = `{"kty":"EC","crv":"P-256","x":"S8FOmrZ3ywj4yyFqt0etAD90U-EnkNaOBSLfQmf7pNg","y":"vMvpDyqFDRHjGfZ1siDOm5LS6xNdR5xTpyoQGLDOX2Q"}`
	test3KeyPublicJSON  = `{"kty":"RSA","n":"uTQER6vUA1RDixS8xsfCRiKUNGRzzyIK0MhbS2biClShbb0hSx2mPP7gBvis2lizZ9r-y9hL57kNQoYCKndOBg0FYsHzrQ3O9AcoV1z2Mq-XhHZbFrVYaXI0M3oY9BJCWog0dyi3XC0x8AxC1npd1U61cToHx-3uSvgZOuQA5ffEn5L38Dz1Ti7OV3E4XahnRJvejadUmTkki7phLBUXm5MnnyFm0CPpf6ApV7zhLjN5W-nV0WL17o7v8aDgV_t9nIdi1Y26c3PlCEtiVHZcebDH5F1Deta3oLLg9-g6rWnTqPbY3knffhp4m0scLD6e33k8MtzxDX_D7vHsg0_X1w","e":"AQAB"}`
	test4KeyPublicJSON  = `{"kty":"RSA","n":"qih-cx32M0wq8MhhN-kBi2xPE-wnw4_iIg1hWO5wtBfpt2PtWikgPuBT6jvK9oyQwAWbSfwqlVZatMPY_-3IyytMNb9R9OatNr6o5HROBoyZnDVSiC4iMRd7bRl_PWSIqj_MjhPNa9cYwBdW5iC3jM5TaOgmp0-YFm4tkLGirDcIBDkQYlnv9NKILvuwqkapZ7XBixeqdCcikUcTRXW5unqygO6bnapzw-YtPsPPlj4Ih3SvK4doyziPV96U8u5lbNYYEzYiW1mbu9n0KLvmKDikGcdOpf6-yRa_10kMZyYQatY1eclIKI0xb54kbluEl0GQDaL5FxLmiKeVnsapzw","e":"AQAB"}`

	agreementURL = "http://example.invalid/terms"
)

// GetRegistration is a mock
func (sa *StorageAuthorityReadOnly) GetRegistration(_ context.Context, req *sapb.RegistrationID, _ ...grpc.CallOption) (*corepb.Registration, error) {
	if req.Id == 100 {
		// Tag meaning "Missing"
		return nil, errors.New("missing")
	}
	if req.Id == 101 {
		// Tag meaning "Malformed"
		return &corepb.Registration{}, nil
	}
	if req.Id == 102 {
		// Tag meaning "Not Found"
		return nil, berrors.NotFoundError("Dave's not here man")
	}

	goodReg := &corepb.Registration{
		Id:              req.Id,
		Key:             []byte(test1KeyPublicJSON),
		Agreement:       agreementURL,
		Contact:         []string{"mailto:person@mail.com"},
		ContactsPresent: true,
		Status:          string(core.StatusValid),
	}

	// Return a populated registration with contacts for ID == 1 or ID == 5
	if req.Id == 1 || req.Id == 5 {
		return goodReg, nil
	}

	// Return a populated registration with a different key for ID == 2
	if req.Id == 2 {
		goodReg.Key = []byte(test2KeyPublicJSON)
		return goodReg, nil
	}

	// Return a deactivated registration with a different key for ID == 3
	if req.Id == 3 {
		goodReg.Key = []byte(test3KeyPublicJSON)
		goodReg.Status = string(core.StatusDeactivated)
		return goodReg, nil
	}

	// Return a populated registration with a different key for ID == 4
	if req.Id == 4 {
		goodReg.Key = []byte(test4KeyPublicJSON)
		return goodReg, nil
	}

	// Return a registration without the agreement set for ID == 6
	if req.Id == 6 {
		goodReg.Agreement = ""
		return goodReg, nil
	}

	goodReg.InitialIP, _ = net.ParseIP("5.6.7.8").MarshalText()
	createdAt := time.Date(2003, 9, 27, 0, 0, 0, 0, time.UTC)
	goodReg.CreatedAt = createdAt.UnixNano()
	return goodReg, nil
}

// GetRegistrationByKey is a mock
func (sa *StorageAuthorityReadOnly) GetRegistrationByKey(_ context.Context, req *sapb.JSONWebKey, _ ...grpc.CallOption) (*corepb.Registration, error) {
	test5KeyBytes, err := os.ReadFile("../test/test-key-5.der")
	if err != nil {
		return nil, err
	}
	test5KeyPriv, err := x509.ParsePKCS1PrivateKey(test5KeyBytes)
	if err != nil {
		return nil, err
	}
	test5KeyPublic := jose.JSONWebKey{Key: test5KeyPriv.Public()}
	test5KeyPublicJSON, err := test5KeyPublic.MarshalJSON()
	if err != nil {
		return nil, err
	}

	contacts := []string{"mailto:person@mail.com"}

	if bytes.Equal(req.Jwk, []byte(test1KeyPublicJSON)) {
		return &corepb.Registration{
			Id:              1,
			Key:             req.Jwk,
			Agreement:       agreementURL,
			Contact:         contacts,
			ContactsPresent: true,
			Status:          string(core.StatusValid),
		}, nil
	}

	if bytes.Equal(req.Jwk, []byte(test2KeyPublicJSON)) {
		// No key found
		return &corepb.Registration{Id: 2}, berrors.NotFoundError("reg not found")
	}

	if bytes.Equal(req.Jwk, []byte(test4KeyPublicJSON)) {
		// No key found
		return &corepb.Registration{Id: 5}, berrors.NotFoundError("reg not found")
	}

	if bytes.Equal(req.Jwk, test5KeyPublicJSON) {
		// No key found
		return &corepb.Registration{Id: 5}, berrors.NotFoundError("reg not found")
	}

	if bytes.Equal(req.Jwk, []byte(testE1KeyPublicJSON)) {
		return &corepb.Registration{Id: 3, Key: req.Jwk, Agreement: agreementURL}, nil
	}

	if bytes.Equal(req.Jwk, []byte(testE2KeyPublicJSON)) {
		return &corepb.Registration{Id: 4}, berrors.NotFoundError("reg not found")
	}

	if bytes.Equal(req.Jwk, []byte(test3KeyPublicJSON)) {
		// deactivated registration
		return &corepb.Registration{
			Id:              2,
			Key:             req.Jwk,
			Agreement:       agreementURL,
			Contact:         contacts,
			ContactsPresent: true,
			Status:          string(core.StatusDeactivated),
		}, nil
	}

	// Return a fake registration. Make sure to fill the key field to avoid marshaling errors.
	return &corepb.Registration{
		Id:        1,
		Key:       []byte(test1KeyPublicJSON),
		Agreement: agreementURL,
		Status:    string(core.StatusValid),
	}, nil
}

// GetSerialMetadata is a mock
func (sa *StorageAuthorityReadOnly) GetSerialMetadata(ctx context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*sapb.SerialMetadata, error) {
	return &sapb.SerialMetadata{
		Serial:         req.Serial,
		RegistrationID: 1,
		Created:        sa.clk.Now().Add(-1 * time.Hour).UnixNano(),
		Expires:        sa.clk.Now().Add(2159 * time.Hour).UnixNano(),
	}, nil
}

// GetCertificate is a mock
func (sa *StorageAuthorityReadOnly) GetCertificate(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.Certificate, error) {
	// Serial ee == 238.crt
	if req.Serial == "0000000000000000000000000000000000ee" {
		certPemBytes, _ := os.ReadFile("test/238.crt")
		certBlock, _ := pem.Decode(certPemBytes)
		return &corepb.Certificate{
			RegistrationID: 1,
			Der:            certBlock.Bytes,
			Issued:         sa.clk.Now().Add(-1 * time.Hour).UnixNano(),
		}, nil
	} else if req.Serial == "0000000000000000000000000000000000b2" {
		certPemBytes, _ := os.ReadFile("test/178.crt")
		certBlock, _ := pem.Decode(certPemBytes)
		return &corepb.Certificate{
			RegistrationID: 1,
			Der:            certBlock.Bytes,
			Issued:         sa.clk.Now().Add(-1 * time.Hour).UnixNano(),
		}, nil
	} else if req.Serial == "000000000000000000000000000000626164" {
		return nil, errors.New("bad")
	} else {
		return nil, berrors.NotFoundError("No cert")
	}
}

// GetPrecertificate is a mock
func (sa *StorageAuthorityReadOnly) GetPrecertificate(_ context.Context, _ *sapb.Serial, _ ...grpc.CallOption) (*corepb.Certificate, error) {
	return nil, nil
}

// GetCertificateStatus is a mock
func (sa *StorageAuthorityReadOnly) GetCertificateStatus(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.CertificateStatus, error) {
	// Serial ee == 238.crt
	if req.Serial == "0000000000000000000000000000000000ee" {
		return &corepb.CertificateStatus{
			Status: string(core.OCSPStatusGood),
		}, nil
	} else if req.Serial == "0000000000000000000000000000000000b2" {
		return &corepb.CertificateStatus{
			Status: string(core.OCSPStatusRevoked),
		}, nil
	} else {
		return nil, errors.New("No cert status")
	}
}

// GetRevocationStatus is a mock
func (sa *StorageAuthorityReadOnly) GetRevocationStatus(_ context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*sapb.RevocationStatus, error) {
	return nil, nil
}

// SerialsForIncident is a mock
func (sa *StorageAuthorityReadOnly) SerialsForIncident(ctx context.Context, _ *sapb.SerialsForIncidentRequest, _ ...grpc.CallOption) (sapb.StorageAuthorityReadOnly_SerialsForIncidentClient, error) {
	return nil, nil
}

// SerialsForIncident is a mock
func (sa *StorageAuthority) SerialsForIncident(ctx context.Context, _ *sapb.SerialsForIncidentRequest, _ ...grpc.CallOption) (sapb.StorageAuthority_SerialsForIncidentClient, error) {
	return nil, nil
}

// GetRevokedCerts is a mock
func (sa *StorageAuthorityReadOnly) GetRevokedCerts(ctx context.Context, _ *sapb.GetRevokedCertsRequest, _ ...grpc.CallOption) (sapb.StorageAuthorityReadOnly_GetRevokedCertsClient, error) {
	return nil, nil
}

// GetRevokedCerts is a mock
func (sa *StorageAuthority) GetRevokedCerts(ctx context.Context, _ *sapb.GetRevokedCertsRequest, _ ...grpc.CallOption) (sapb.StorageAuthority_GetRevokedCertsClient, error) {
	return nil, nil
}

// GetMaxExpiration is a mock
func (sa *StorageAuthorityReadOnly) GetMaxExpiration(_ context.Context, req *emptypb.Empty, _ ...grpc.CallOption) (*timestamppb.Timestamp, error) {
	return nil, nil
}

// AddPrecertificate is a mock
func (sa *StorageAuthority) AddPrecertificate(ctx context.Context, req *sapb.AddCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

// AddSerial is a mock
func (sa *StorageAuthority) AddSerial(ctx context.Context, req *sapb.AddSerialRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

// AddCertificate is a mock
func (sa *StorageAuthority) AddCertificate(_ context.Context, _ *sapb.AddCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

// NewRegistration is a mock
func (sa *StorageAuthority) NewRegistration(_ context.Context, _ *corepb.Registration, _ ...grpc.CallOption) (*corepb.Registration, error) {
	return &corepb.Registration{}, nil
}

// UpdateRegistration is a mock
func (sa *StorageAuthority) UpdateRegistration(_ context.Context, _ *corepb.Registration, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// CountFQDNSets is a mock
func (sa *StorageAuthorityReadOnly) CountFQDNSets(_ context.Context, _ *sapb.CountFQDNSetsRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return &sapb.Count{}, nil
}

// FQDNSetIssuanceForWindow is a mock
func (sa *StorageAuthorityReadOnly) FQDNSetTimestampsForWindow(_ context.Context, _ *sapb.CountFQDNSetsRequest, _ ...grpc.CallOption) (*sapb.Timestamps, error) {
	return &sapb.Timestamps{}, nil
}

// FQDNSetExists is a mock
func (sa *StorageAuthorityReadOnly) FQDNSetExists(_ context.Context, _ *sapb.FQDNSetExistsRequest, _ ...grpc.CallOption) (*sapb.Exists, error) {
	return &sapb.Exists{Exists: false}, nil
}

func (sa *StorageAuthorityReadOnly) PreviousCertificateExists(_ context.Context, _ *sapb.PreviousCertificateExistsRequest, _ ...grpc.CallOption) (*sapb.Exists, error) {
	return &sapb.Exists{Exists: false}, nil
}

// CountCertificatesByNames is a mock
func (sa *StorageAuthorityReadOnly) CountCertificatesByNames(_ context.Context, _ *sapb.CountCertificatesByNamesRequest, _ ...grpc.CallOption) (*sapb.CountByNames, error) {
	return &sapb.CountByNames{}, nil
}

// CountRegistrationsByIP is a mock
func (sa *StorageAuthorityReadOnly) CountRegistrationsByIP(_ context.Context, _ *sapb.CountRegistrationsByIPRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return &sapb.Count{}, nil
}

// CountRegistrationsByIPRange is a mock
func (sa *StorageAuthorityReadOnly) CountRegistrationsByIPRange(_ context.Context, _ *sapb.CountRegistrationsByIPRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return &sapb.Count{}, nil
}

// CountOrders is a mock
func (sa *StorageAuthorityReadOnly) CountOrders(_ context.Context, _ *sapb.CountOrdersRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return &sapb.Count{}, nil
}

// DeactivateRegistration is a mock
func (sa *StorageAuthority) DeactivateRegistration(_ context.Context, _ *sapb.RegistrationID, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// NewOrderAndAuthzs is a mock
func (sa *StorageAuthority) NewOrderAndAuthzs(_ context.Context, req *sapb.NewOrderAndAuthzsRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	rand.Seed(time.Now().UnixNano())
	response := &corepb.Order{
		// Fields from the input new order request.
		RegistrationID:   req.NewOrder.RegistrationID,
		Expires:          req.NewOrder.Expires,
		Names:            req.NewOrder.Names,
		V2Authorizations: req.NewOrder.V2Authorizations,
		// Mock new fields generated by the database transaction.
		Id:      rand.Int63(),
		Created: time.Now().UnixNano(),
		// A new order is never processing because it can't have been finalized yet.
		BeganProcessing: false,
		Status:          string(core.StatusPending),
	}
	return response, nil
}

// SetOrderProcessing is a mock
func (sa *StorageAuthority) SetOrderProcessing(_ context.Context, req *sapb.OrderRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// SetOrderError is a mock
func (sa *StorageAuthority) SetOrderError(_ context.Context, req *sapb.SetOrderErrorRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// FinalizeOrder is a mock
func (sa *StorageAuthority) FinalizeOrder(_ context.Context, req *sapb.FinalizeOrderRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// GetOrder is a mock
func (sa *StorageAuthorityReadOnly) GetOrder(_ context.Context, req *sapb.OrderRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	if req.Id == 2 {
		return nil, berrors.NotFoundError("bad")
	} else if req.Id == 3 {
		return nil, errors.New("very bad")
	}

	created := sa.clk.Now().AddDate(-30, 0, 0).Unix()
	exp := sa.clk.Now().AddDate(30, 0, 0).Unix()
	validOrder := &corepb.Order{
		Id:                req.Id,
		RegistrationID:    1,
		Created:           created,
		Expires:           exp,
		Names:             []string{"example.com"},
		Status:            string(core.StatusValid),
		V2Authorizations:  []int64{1},
		CertificateSerial: "serial",
		Error:             nil,
	}

	// Order ID doesn't have a certificate serial yet
	if req.Id == 4 {
		validOrder.Status = string(core.StatusPending)
		validOrder.Id = req.Id
		validOrder.CertificateSerial = ""
		validOrder.Error = nil
		return validOrder, nil
	}

	// Order ID 6 belongs to reg ID 6
	if req.Id == 6 {
		validOrder.Id = 6
		validOrder.RegistrationID = 6
	}

	// Order ID 7 is ready, but expired
	if req.Id == 7 {
		validOrder.Status = string(core.StatusReady)
		validOrder.Expires = sa.clk.Now().AddDate(-30, 0, 0).Unix()
	}

	if req.Id == 8 {
		validOrder.Status = string(core.StatusReady)
	}

	// Order 9 is fresh
	if req.Id == 9 {
		validOrder.Created = sa.clk.Now().AddDate(0, 0, 1).Unix()
	}

	return validOrder, nil
}

func (sa *StorageAuthorityReadOnly) GetOrderForNames(_ context.Context, _ *sapb.GetOrderForNamesRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	return nil, nil
}

func (sa *StorageAuthority) FinalizeAuthorization2(ctx context.Context, req *sapb.FinalizeAuthorizationRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (sa *StorageAuthority) DeactivateAuthorization2(ctx context.Context, req *sapb.AuthorizationID2, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (sa *StorageAuthorityReadOnly) CountPendingAuthorizations2(ctx context.Context, req *sapb.RegistrationID, _ ...grpc.CallOption) (*sapb.Count, error) {
	return &sapb.Count{}, nil
}

func (sa *StorageAuthorityReadOnly) GetValidOrderAuthorizations2(ctx context.Context, req *sapb.GetValidOrderAuthorizationsRequest, _ ...grpc.CallOption) (*sapb.Authorizations, error) {
	return nil, nil
}

func (sa *StorageAuthorityReadOnly) CountInvalidAuthorizations2(ctx context.Context, req *sapb.CountInvalidAuthorizationsRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return &sapb.Count{}, nil
}

func (sa *StorageAuthorityReadOnly) GetValidAuthorizations2(ctx context.Context, req *sapb.GetValidAuthorizationsRequest, _ ...grpc.CallOption) (*sapb.Authorizations, error) {
	if req.RegistrationID != 1 && req.RegistrationID != 5 && req.RegistrationID != 4 {
		return &sapb.Authorizations{}, nil
	}
	now := time.Unix(0, req.Now)
	auths := &sapb.Authorizations{}
	for _, name := range req.Domains {
		exp := now.AddDate(100, 0, 0)
		authzPB, err := bgrpc.AuthzToPB(core.Authorization{
			Status:         core.StatusValid,
			RegistrationID: req.RegistrationID,
			Expires:        &exp,
			Identifier: identifier.ACMEIdentifier{
				Type:  "dns",
				Value: name,
			},
			Challenges: []core.Challenge{
				{
					Status:    core.StatusValid,
					Type:      core.ChallengeTypeDNS01,
					Token:     "exampleToken",
					Validated: &now,
				},
			},
		})
		if err != nil {
			return nil, err
		}
		auths.Authz = append(auths.Authz, &sapb.Authorizations_MapElement{
			Domain: name,
			Authz:  authzPB,
		})
	}
	return auths, nil
}

func (sa *StorageAuthorityReadOnly) GetAuthorizations2(ctx context.Context, req *sapb.GetAuthorizationsRequest, _ ...grpc.CallOption) (*sapb.Authorizations, error) {
	return &sapb.Authorizations{}, nil
}

func (sa *StorageAuthorityReadOnly) GetPendingAuthorization2(ctx context.Context, req *sapb.GetPendingAuthorizationRequest, _ ...grpc.CallOption) (*corepb.Authorization, error) {
	return nil, nil
}

var (
	authzIdValid       = int64(1)
	authzIdPending     = int64(2)
	authzIdExpired     = int64(3)
	authzIdErrorResult = int64(4)
	authzIdDiffAccount = int64(5)
)

// GetAuthorization2 is a mock
func (sa *StorageAuthorityReadOnly) GetAuthorization2(ctx context.Context, id *sapb.AuthorizationID2, _ ...grpc.CallOption) (*corepb.Authorization, error) {
	authz := core.Authorization{
		Status:         core.StatusValid,
		RegistrationID: 1,
		Identifier:     identifier.DNSIdentifier("not-an-example.com"),
		Challenges: []core.Challenge{
			{
				Status: "pending",
				Token:  "token",
				Type:   "dns",
			},
		},
	}

	switch id.Id {
	case authzIdValid:
		exp := sa.clk.Now().AddDate(100, 0, 0)
		authz.Expires = &exp
		authz.ID = fmt.Sprintf("%d", authzIdValid)
		return bgrpc.AuthzToPB(authz)
	case authzIdPending:
		exp := sa.clk.Now().AddDate(100, 0, 0)
		authz.Expires = &exp
		authz.ID = fmt.Sprintf("%d", authzIdPending)
		authz.Status = core.StatusPending
		return bgrpc.AuthzToPB(authz)
	case authzIdExpired:
		exp := sa.clk.Now().AddDate(0, -1, 0)
		authz.Expires = &exp
		authz.ID = fmt.Sprintf("%d", authzIdExpired)
		return bgrpc.AuthzToPB(authz)
	case authzIdErrorResult:
		return nil, fmt.Errorf("Unspecified database error")
	case authzIdDiffAccount:
		exp := sa.clk.Now().AddDate(100, 0, 0)
		authz.RegistrationID = 2
		authz.Expires = &exp
		authz.ID = fmt.Sprintf("%d", authzIdDiffAccount)
		return bgrpc.AuthzToPB(authz)
	}

	return nil, berrors.NotFoundError("no authorization found with id %q", id)
}

// RevokeCertificate is a mock
func (sa *StorageAuthority) RevokeCertificate(ctx context.Context, req *sapb.RevokeCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

// RevokeCertificate is a mock
func (sa *StorageAuthority) UpdateRevokedCertificate(ctx context.Context, req *sapb.RevokeCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

// AddBlockedKey is a mock
func (sa *StorageAuthority) AddBlockedKey(ctx context.Context, req *sapb.AddBlockedKeyRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// KeyBlocked is a mock
func (sa *StorageAuthorityReadOnly) KeyBlocked(ctx context.Context, req *sapb.KeyBlockedRequest, _ ...grpc.CallOption) (*sapb.Exists, error) {
	return &sapb.Exists{Exists: false}, nil
}

// IncidentsForSerial is a mock.
func (sa *StorageAuthorityReadOnly) IncidentsForSerial(ctx context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*sapb.Incidents, error) {
	return &sapb.Incidents{}, nil
}

// Publisher is a mock
type PublisherClient struct {
	// empty
}

// SubmitToSingleCTWithResult is a mock
func (*PublisherClient) SubmitToSingleCTWithResult(_ context.Context, _ *pubpb.Request, _ ...grpc.CallOption) (*pubpb.Result, error) {
	return &pubpb.Result{}, nil
}

// Mailer is a mock
type Mailer struct {
	sync.Mutex
	Messages []MailerMessage
}

var _ mail.Mailer = &Mailer{}

// mockMailerConn is a mock that satisfies the mail.Conn interface
type mockMailerConn struct {
	parent *Mailer
}

var _ mail.Conn = &mockMailerConn{}

// MailerMessage holds the captured emails from SendMail()
type MailerMessage struct {
	To      string
	Subject string
	Body    string
}

// Clear removes any previously recorded messages
func (m *Mailer) Clear() {
	m.Lock()
	defer m.Unlock()
	m.Messages = nil
}

// SendMail is a mock
func (m *mockMailerConn) SendMail(to []string, subject, msg string) error {
	m.parent.Lock()
	defer m.parent.Unlock()
	for _, rcpt := range to {
		m.parent.Messages = append(m.parent.Messages, MailerMessage{
			To:      rcpt,
			Subject: subject,
			Body:    msg,
		})
	}
	return nil
}

// Close is a mock
func (m *mockMailerConn) Close() error {
	return nil
}

// Connect is a mock
func (m *Mailer) Connect() (mail.Conn, error) {
	return &mockMailerConn{parent: m}, nil
}

// SAWithFailedChallenges is a mocks.StorageAuthority that has
// a `GetAuthorization` implementation that can return authorizations with
// failed challenges.
type SAWithFailedChallenges struct {
	StorageAuthorityReadOnly
	Clk clock.FakeClock
}

func (sa *SAWithFailedChallenges) GetAuthorization2(ctx context.Context, id *sapb.AuthorizationID2, _ ...grpc.CallOption) (*corepb.Authorization, error) {
	authz := core.Authorization{
		ID:             "55",
		Status:         core.StatusValid,
		RegistrationID: 1,
		Identifier:     identifier.DNSIdentifier("not-an-example.com"),
		Challenges: []core.Challenge{
			{
				Status: core.StatusInvalid,
				Type:   "dns",
				Token:  "exampleToken",
			},
		},
	}
	prob := &probs.ProblemDetails{
		Type:       "things:are:whack",
		Detail:     "whack attack",
		HTTPStatus: 555,
	}
	exp := sa.Clk.Now().AddDate(100, 0, 0)
	authz.Expires = &exp
	// 55 returns an authz with a failed challenge that has the problem type
	// statically prefixed by the V1ErrorNS
	if id.Id == 55 {
		prob.Type = probs.V1ErrorNS + prob.Type
		authz.Challenges[0].Error = prob
		return bgrpc.AuthzToPB(authz)
	}
	// 56 returns an authz with a failed challenge that has no error
	// namespace on the problem type.
	if id.Id == 56 {
		authz.Challenges[0].Error = prob
		return bgrpc.AuthzToPB(authz)
	}
	return nil, berrors.NotFoundError("no authorization found with id %q", id)
}
