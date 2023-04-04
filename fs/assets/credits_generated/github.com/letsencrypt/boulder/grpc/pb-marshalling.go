// Copyright 2016 ISRG.  All rights reserved
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package grpc

import (
	"net"
	"time"

	"google.golang.org/grpc/codes"
	"gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/probs"
	"github.com/letsencrypt/boulder/revocation"
	sapb "github.com/letsencrypt/boulder/sa/proto"
	vapb "github.com/letsencrypt/boulder/va/proto"
)

var ErrMissingParameters = CodedError(codes.FailedPrecondition, "required RPC parameter was missing")

// This file defines functions to translate between the protobuf types and the
// code types.

func ProblemDetailsToPB(prob *probs.ProblemDetails) (*corepb.ProblemDetails, error) {
	if prob == nil {
		// nil problemDetails is valid
		return nil, nil
	}
	return &corepb.ProblemDetails{
		ProblemType: string(prob.Type),
		Detail:      prob.Detail,
		HttpStatus:  int32(prob.HTTPStatus),
	}, nil
}

func PBToProblemDetails(in *corepb.ProblemDetails) (*probs.ProblemDetails, error) {
	if in == nil {
		// nil problemDetails is valid
		return nil, nil
	}
	if in.ProblemType == "" || in.Detail == "" {
		return nil, ErrMissingParameters
	}
	prob := &probs.ProblemDetails{
		Type:   probs.ProblemType(in.ProblemType),
		Detail: in.Detail,
	}
	if in.HttpStatus != 0 {
		prob.HTTPStatus = int(in.HttpStatus)
	}
	return prob, nil
}

func ChallengeToPB(challenge core.Challenge) (*corepb.Challenge, error) {
	prob, err := ProblemDetailsToPB(challenge.Error)
	if err != nil {
		return nil, err
	}
	recordAry := make([]*corepb.ValidationRecord, len(challenge.ValidationRecord))
	for i, v := range challenge.ValidationRecord {
		recordAry[i], err = ValidationRecordToPB(v)
		if err != nil {
			return nil, err
		}
	}
	var validated int64
	if challenge.Validated != nil {
		validated = challenge.Validated.UTC().UnixNano()
	}
	return &corepb.Challenge{
		Type:              string(challenge.Type),
		Status:            string(challenge.Status),
		Token:             challenge.Token,
		KeyAuthorization:  challenge.ProvidedKeyAuthorization,
		Error:             prob,
		Validationrecords: recordAry,
		Validated:         validated,
	}, nil
}

func PBToChallenge(in *corepb.Challenge) (challenge core.Challenge, err error) {
	if in == nil {
		return core.Challenge{}, ErrMissingParameters
	}
	if in.Type == "" || in.Status == "" || in.Token == "" {
		return core.Challenge{}, ErrMissingParameters
	}
	var recordAry []core.ValidationRecord
	if len(in.Validationrecords) > 0 {
		recordAry = make([]core.ValidationRecord, len(in.Validationrecords))
		for i, v := range in.Validationrecords {
			recordAry[i], err = PBToValidationRecord(v)
			if err != nil {
				return core.Challenge{}, err
			}
		}
	}
	prob, err := PBToProblemDetails(in.Error)
	if err != nil {
		return core.Challenge{}, err
	}
	var validated *time.Time
	if in.Validated != 0 {
		val := time.Unix(0, in.Validated).UTC()
		validated = &val
	}
	ch := core.Challenge{
		Type:             core.AcmeChallenge(in.Type),
		Status:           core.AcmeStatus(in.Status),
		Token:            in.Token,
		Error:            prob,
		ValidationRecord: recordAry,
		Validated:        validated,
	}
	if in.KeyAuthorization != "" {
		ch.ProvidedKeyAuthorization = in.KeyAuthorization
	}
	return ch, nil
}

func ValidationRecordToPB(record core.ValidationRecord) (*corepb.ValidationRecord, error) {
	addrs := make([][]byte, len(record.AddressesResolved))
	addrsTried := make([][]byte, len(record.AddressesTried))
	var err error
	for i, v := range record.AddressesResolved {
		addrs[i] = []byte(v)
	}
	for i, v := range record.AddressesTried {
		addrsTried[i] = []byte(v)
	}
	addrUsed, err := record.AddressUsed.MarshalText()
	if err != nil {
		return nil, err
	}
	return &corepb.ValidationRecord{
		Hostname:          record.Hostname,
		Port:              record.Port,
		AddressesResolved: addrs,
		AddressUsed:       addrUsed,
		Url:               record.URL,
		AddressesTried:    addrsTried,
	}, nil
}

func PBToValidationRecord(in *corepb.ValidationRecord) (record core.ValidationRecord, err error) {
	if in == nil {
		return core.ValidationRecord{}, ErrMissingParameters
	}
	addrs := make([]net.IP, len(in.AddressesResolved))
	for i, v := range in.AddressesResolved {
		addrs[i] = net.IP(v)
	}
	addrsTried := make([]net.IP, len(in.AddressesTried))
	for i, v := range in.AddressesTried {
		addrsTried[i] = net.IP(v)
	}
	var addrUsed net.IP
	err = addrUsed.UnmarshalText(in.AddressUsed)
	if err != nil {
		return
	}
	return core.ValidationRecord{
		Hostname:          in.Hostname,
		Port:              in.Port,
		AddressesResolved: addrs,
		AddressUsed:       addrUsed,
		URL:               in.Url,
		AddressesTried:    addrsTried,
	}, nil
}

func ValidationResultToPB(records []core.ValidationRecord, prob *probs.ProblemDetails) (*vapb.ValidationResult, error) {
	recordAry := make([]*corepb.ValidationRecord, len(records))
	var err error
	for i, v := range records {
		recordAry[i], err = ValidationRecordToPB(v)
		if err != nil {
			return nil, err
		}
	}
	marshalledProbs, err := ProblemDetailsToPB(prob)
	if err != nil {
		return nil, err
	}
	return &vapb.ValidationResult{
		Records:  recordAry,
		Problems: marshalledProbs,
	}, nil
}

func pbToValidationResult(in *vapb.ValidationResult) ([]core.ValidationRecord, *probs.ProblemDetails, error) {
	if in == nil {
		return nil, nil, ErrMissingParameters
	}
	recordAry := make([]core.ValidationRecord, len(in.Records))
	var err error
	for i, v := range in.Records {
		recordAry[i], err = PBToValidationRecord(v)
		if err != nil {
			return nil, nil, err
		}
	}
	prob, err := PBToProblemDetails(in.Problems)
	if err != nil {
		return nil, nil, err
	}
	return recordAry, prob, nil
}

func RegistrationToPB(reg core.Registration) (*corepb.Registration, error) {
	keyBytes, err := reg.Key.MarshalJSON()
	if err != nil {
		return nil, err
	}
	ipBytes, err := reg.InitialIP.MarshalText()
	if err != nil {
		return nil, err
	}
	var contacts []string
	// Since the default value of corepb.Registration.Contact is a slice
	// we need a indicator as to if the value is actually important on
	// the other side (pb -> reg).
	contactsPresent := reg.Contact != nil
	if reg.Contact != nil {
		contacts = *reg.Contact
	}
	var createdAt int64
	if reg.CreatedAt != nil {
		createdAt = reg.CreatedAt.UTC().UnixNano()
	}
	return &corepb.Registration{
		Id:              reg.ID,
		Key:             keyBytes,
		Contact:         contacts,
		ContactsPresent: contactsPresent,
		Agreement:       reg.Agreement,
		InitialIP:       ipBytes,
		CreatedAt:       createdAt,
		Status:          string(reg.Status),
	}, nil
}

func PbToRegistration(pb *corepb.Registration) (core.Registration, error) {
	var key jose.JSONWebKey
	err := key.UnmarshalJSON(pb.Key)
	if err != nil {
		return core.Registration{}, err
	}
	var initialIP net.IP
	err = initialIP.UnmarshalText(pb.InitialIP)
	if err != nil {
		return core.Registration{}, err
	}
	var createdAt *time.Time
	if pb.CreatedAt != 0 {
		c := time.Unix(0, pb.CreatedAt).UTC()
		createdAt = &c
	}
	var contacts *[]string
	if pb.ContactsPresent {
		if len(pb.Contact) != 0 {
			contacts = &pb.Contact
		} else {
			// When gRPC creates an empty slice it is actually a nil slice. Since
			// certain things boulder uses, like encoding/json, differentiate between
			// these we need to de-nil these slices. Without this we are unable to
			// properly do registration updates as contacts would always be removed
			// as we use the difference between a nil and empty slice in ra.mergeUpdate.
			empty := []string{}
			contacts = &empty
		}
	}
	return core.Registration{
		ID:        pb.Id,
		Key:       &key,
		Contact:   contacts,
		Agreement: pb.Agreement,
		InitialIP: initialIP,
		CreatedAt: createdAt,
		Status:    core.AcmeStatus(pb.Status),
	}, nil
}

func AuthzToPB(authz core.Authorization) (*corepb.Authorization, error) {
	challs := make([]*corepb.Challenge, len(authz.Challenges))
	for i, c := range authz.Challenges {
		pbChall, err := ChallengeToPB(c)
		if err != nil {
			return nil, err
		}
		challs[i] = pbChall
	}
	var expires int64
	if authz.Expires != nil {
		expires = authz.Expires.UTC().UnixNano()
	}
	return &corepb.Authorization{
		Id:             authz.ID,
		Identifier:     authz.Identifier.Value,
		RegistrationID: authz.RegistrationID,
		Status:         string(authz.Status),
		Expires:        expires,
		Challenges:     challs,
	}, nil
}

func PBToAuthz(pb *corepb.Authorization) (core.Authorization, error) {
	challs := make([]core.Challenge, len(pb.Challenges))
	for i, c := range pb.Challenges {
		chall, err := PBToChallenge(c)
		if err != nil {
			return core.Authorization{}, err
		}
		challs[i] = chall
	}
	expires := time.Unix(0, pb.Expires).UTC()
	authz := core.Authorization{
		ID:             pb.Id,
		Identifier:     identifier.ACMEIdentifier{Type: identifier.DNS, Value: pb.Identifier},
		RegistrationID: pb.RegistrationID,
		Status:         core.AcmeStatus(pb.Status),
		Expires:        &expires,
		Challenges:     challs,
	}
	return authz, nil
}

// orderValid checks that a corepb.Order is valid. In addition to the checks
// from `newOrderValid` it ensures the order ID and the Created field are not nil.
func orderValid(order *corepb.Order) bool {
	return order.Id != 0 && order.Created != 0 && newOrderValid(order)
}

// newOrderValid checks that a corepb.Order is valid. It allows for a nil
// `order.Id` because the order has not been assigned an ID yet when it is being
// created initially. It allows `order.BeganProcessing` to be nil because
// `sa.NewOrder` explicitly sets it to the default value. It allows
// `order.Created` to be nil because the SA populates this. It also allows
// `order.CertificateSerial` to be nil such that it can be used in places where
// the order has not been finalized yet.
func newOrderValid(order *corepb.Order) bool {
	return !(order.RegistrationID == 0 || order.Expires == 0 || len(order.Names) == 0)
}

func CertToPB(cert core.Certificate) *corepb.Certificate {
	return &corepb.Certificate{
		RegistrationID: cert.RegistrationID,
		Serial:         cert.Serial,
		Digest:         cert.Digest,
		Der:            cert.DER,
		Issued:         cert.Issued.UnixNano(),
		Expires:        cert.Expires.UnixNano(),
	}
}

func PBToCert(pb *corepb.Certificate) (core.Certificate, error) {
	return core.Certificate{
		RegistrationID: pb.RegistrationID,
		Serial:         pb.Serial,
		Digest:         pb.Digest,
		DER:            pb.Der,
		Issued:         time.Unix(0, pb.Issued),
		Expires:        time.Unix(0, pb.Expires),
	}, nil
}

func CertStatusToPB(certStatus core.CertificateStatus) *corepb.CertificateStatus {
	return &corepb.CertificateStatus{
		Serial:                certStatus.Serial,
		Status:                string(certStatus.Status),
		OcspLastUpdated:       certStatus.OCSPLastUpdated.UnixNano(),
		RevokedDate:           certStatus.RevokedDate.UnixNano(),
		RevokedReason:         int64(certStatus.RevokedReason),
		LastExpirationNagSent: certStatus.LastExpirationNagSent.UnixNano(),
		OcspResponse:          certStatus.OCSPResponse,
		NotAfter:              certStatus.NotAfter.UnixNano(),
		IsExpired:             certStatus.IsExpired,
		IssuerID:              certStatus.IssuerID,
	}
}

func PBToCertStatus(pb *corepb.CertificateStatus) (core.CertificateStatus, error) {
	return core.CertificateStatus{
		Serial:                pb.Serial,
		Status:                core.OCSPStatus(pb.Status),
		OCSPLastUpdated:       time.Unix(0, pb.OcspLastUpdated),
		RevokedDate:           time.Unix(0, pb.RevokedDate),
		RevokedReason:         revocation.Reason(pb.RevokedReason),
		LastExpirationNagSent: time.Unix(0, pb.LastExpirationNagSent),
		OCSPResponse:          pb.OcspResponse,
		NotAfter:              time.Unix(0, pb.NotAfter),
		IsExpired:             pb.IsExpired,
		IssuerID:              pb.IssuerID,
	}, nil
}

// PBToAuthzMap converts a protobuf map of domains mapped to protobuf authorizations to a
// golang map[string]*core.Authorization.
func PBToAuthzMap(pb *sapb.Authorizations) (map[string]*core.Authorization, error) {
	m := make(map[string]*core.Authorization, len(pb.Authz))
	for _, v := range pb.Authz {
		authz, err := PBToAuthz(v.Authz)
		if err != nil {
			return nil, err
		}
		m[v.Domain] = &authz
	}
	return m, nil
}
