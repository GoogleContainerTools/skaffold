package sa

import (
	"context"
	"io"

	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/sa"
	sapb "github.com/letsencrypt/boulder/sa/proto"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

// SA meets the `sapb.StorageAuthorityClient` interface and acts as a
// wrapper for an inner `sa.SQLStorageAuthority` (which in turn meets
// the `sapb.StorageAuthorityServer` interface). Only methods used by
// unit tests need to be implemented.
type SA struct {
	sapb.StorageAuthorityClient
	Impl *sa.SQLStorageAuthority
}

func (sa SA) NewRegistration(ctx context.Context, req *corepb.Registration, _ ...grpc.CallOption) (*corepb.Registration, error) {
	return sa.Impl.NewRegistration(ctx, req)
}

func (sa SA) GetRegistration(ctx context.Context, req *sapb.RegistrationID, _ ...grpc.CallOption) (*corepb.Registration, error) {
	return sa.Impl.GetRegistration(ctx, req)
}

func (sa SA) CountRegistrationsByIP(ctx context.Context, req *sapb.CountRegistrationsByIPRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return sa.Impl.CountRegistrationsByIP(ctx, req)
}

func (sa SA) CountRegistrationsByIPRange(ctx context.Context, req *sapb.CountRegistrationsByIPRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return sa.Impl.CountRegistrationsByIPRange(ctx, req)
}

func (sa SA) DeactivateRegistration(ctx context.Context, req *sapb.RegistrationID, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.DeactivateRegistration(ctx, req)
}

func (sa SA) GetAuthorization2(ctx context.Context, req *sapb.AuthorizationID2, _ ...grpc.CallOption) (*corepb.Authorization, error) {
	return sa.Impl.GetAuthorization2(ctx, req)
}

func (sa SA) GetAuthorizations2(ctx context.Context, req *sapb.GetAuthorizationsRequest, _ ...grpc.CallOption) (*sapb.Authorizations, error) {
	return sa.Impl.GetAuthorizations2(ctx, req)
}

func (sa SA) GetPendingAuthorization2(ctx context.Context, req *sapb.GetPendingAuthorizationRequest, _ ...grpc.CallOption) (*corepb.Authorization, error) {
	return sa.Impl.GetPendingAuthorization2(ctx, req)
}

func (sa SA) GetValidAuthorizations2(ctx context.Context, req *sapb.GetValidAuthorizationsRequest, _ ...grpc.CallOption) (*sapb.Authorizations, error) {
	return sa.Impl.GetValidAuthorizations2(ctx, req)
}

func (sa SA) GetValidOrderAuthorizations2(ctx context.Context, req *sapb.GetValidOrderAuthorizationsRequest, _ ...grpc.CallOption) (*sapb.Authorizations, error) {
	return sa.Impl.GetValidOrderAuthorizations2(ctx, req)
}

func (sa SA) CountPendingAuthorizations2(ctx context.Context, req *sapb.RegistrationID, _ ...grpc.CallOption) (*sapb.Count, error) {
	return sa.Impl.CountPendingAuthorizations2(ctx, req)
}

func (sa SA) DeactivateAuthorization2(ctx context.Context, req *sapb.AuthorizationID2, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.DeactivateAuthorization2(ctx, req)
}

func (sa SA) FinalizeAuthorization2(ctx context.Context, req *sapb.FinalizeAuthorizationRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.FinalizeAuthorization2(ctx, req)
}

func (sa SA) NewOrderAndAuthzs(ctx context.Context, req *sapb.NewOrderAndAuthzsRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	return sa.Impl.NewOrderAndAuthzs(ctx, req)
}

func (sa SA) GetOrder(ctx context.Context, req *sapb.OrderRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	return sa.Impl.GetOrder(ctx, req)
}

func (sa SA) GetOrderForNames(ctx context.Context, req *sapb.GetOrderForNamesRequest, _ ...grpc.CallOption) (*corepb.Order, error) {
	return sa.Impl.GetOrderForNames(ctx, req)
}

func (sa SA) CountOrders(ctx context.Context, req *sapb.CountOrdersRequest, _ ...grpc.CallOption) (*sapb.Count, error) {
	return sa.Impl.CountOrders(ctx, req)
}

func (sa SA) SetOrderError(ctx context.Context, req *sapb.SetOrderErrorRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.SetOrderError(ctx, req)
}

func (sa SA) SetOrderProcessing(ctx context.Context, req *sapb.OrderRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.SetOrderProcessing(ctx, req)
}

func (sa SA) FinalizeOrder(ctx context.Context, req *sapb.FinalizeOrderRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.FinalizeOrder(ctx, req)
}

func (sa SA) AddPrecertificate(ctx context.Context, req *sapb.AddCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.AddPrecertificate(ctx, req)
}

func (sa SA) CountCertificatesByNames(ctx context.Context, req *sapb.CountCertificatesByNamesRequest, _ ...grpc.CallOption) (*sapb.CountByNames, error) {
	return sa.Impl.CountCertificatesByNames(ctx, req)
}

func (sa SA) RevokeCertificate(ctx context.Context, req *sapb.RevokeCertificateRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.RevokeCertificate(ctx, req)
}

func (sa SA) GetCertificateStatus(ctx context.Context, req *sapb.Serial, _ ...grpc.CallOption) (*corepb.CertificateStatus, error) {
	return sa.Impl.GetCertificateStatus(ctx, req)
}

func (sa SA) AddBlockedKey(ctx context.Context, req *sapb.AddBlockedKeyRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return sa.Impl.AddBlockedKey(ctx, req)
}

func (sa SA) FQDNSetExists(ctx context.Context, req *sapb.FQDNSetExistsRequest, _ ...grpc.CallOption) (*sapb.Exists, error) {
	return sa.Impl.FQDNSetExists(ctx, req)
}

type mockSerialsForIncidentStream_Result struct {
	serial *sapb.IncidentSerial
	err    error
}

type mockSerialsForIncidentStream_Client struct {
	grpc.ClientStream
	stream <-chan mockSerialsForIncidentStream_Result
}

func (c mockSerialsForIncidentStream_Client) Recv() (*sapb.IncidentSerial, error) {
	sfiData := <-c.stream
	return sfiData.serial, sfiData.err
}

type mockSerialsForIncidentStream_Server struct {
	grpc.ServerStream
	context context.Context
	stream  chan<- mockSerialsForIncidentStream_Result
}

func (s mockSerialsForIncidentStream_Server) Send(serial *sapb.IncidentSerial) error {
	s.stream <- mockSerialsForIncidentStream_Result{serial, nil}
	return nil
}

func (s mockSerialsForIncidentStream_Server) Context() context.Context {
	return s.context
}

func (sa SA) SerialsForIncident(ctx context.Context, req *sapb.SerialsForIncidentRequest, _ ...grpc.CallOption) (sapb.StorageAuthority_SerialsForIncidentClient, error) {
	streamChan := make(chan mockSerialsForIncidentStream_Result)
	client := mockSerialsForIncidentStream_Client{stream: streamChan}
	server := mockSerialsForIncidentStream_Server{context: ctx, stream: streamChan}
	go func() {
		err := sa.Impl.SerialsForIncident(req, server)
		if err != nil {
			streamChan <- mockSerialsForIncidentStream_Result{nil, err}
		}
		streamChan <- mockSerialsForIncidentStream_Result{nil, io.EOF}
		close(streamChan)
	}()
	return client, nil
}
