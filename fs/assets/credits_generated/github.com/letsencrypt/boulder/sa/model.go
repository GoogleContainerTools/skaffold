package sa

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	jose "gopkg.in/go-jose/go-jose.v2"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/db"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/grpc"
	"github.com/letsencrypt/boulder/identifier"
	"github.com/letsencrypt/boulder/probs"
	"github.com/letsencrypt/boulder/revocation"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

// errBadJSON is an error type returned when a json.Unmarshal performed by the
// SA fails. It includes both the Unmarshal error and the original JSON data in
// its error message to make it easier to track down the bad JSON data.
type errBadJSON struct {
	msg  string
	json []byte
	err  error
}

// Error returns an error message that includes the json.Unmarshal error as well
// as the bad JSON data.
func (e errBadJSON) Error() string {
	return fmt.Sprintf(
		"%s: error unmarshaling JSON %q: %s",
		e.msg,
		string(e.json),
		e.err)
}

// badJSONError is a convenience function for constructing a errBadJSON instance
// with the provided args.
func badJSONError(msg string, jsonData []byte, err error) error {
	return errBadJSON{
		msg:  msg,
		json: jsonData,
		err:  err,
	}
}

const regFields = "id, jwk, jwk_sha256, contact, agreement, initialIP, createdAt, LockCol, status"

// selectRegistration selects all fields of one registration model
func selectRegistration(s db.OneSelector, q string, args ...interface{}) (*regModel, error) {
	var model regModel
	err := s.SelectOne(
		&model,
		"SELECT "+regFields+" FROM registrations "+q,
		args...,
	)
	return &model, err
}

const certFields = "registrationID, serial, digest, der, issued, expires"

// SelectCertificate selects all fields of one certificate object identified by
// a serial. If more than one row contains the same serial only the first is
// returned.
func SelectCertificate(s db.OneSelector, serial string) (core.Certificate, error) {
	var model core.Certificate
	err := s.SelectOne(
		&model,
		"SELECT "+certFields+" FROM certificates WHERE serial = ? LIMIT 1",
		serial,
	)
	return model, err
}

const precertFields = "registrationID, serial, der, issued, expires"

// SelectPrecertificate selects all fields of one precertificate object
// identified by serial.
func SelectPrecertificate(s db.OneSelector, serial string) (core.Certificate, error) {
	var model precertificateModel
	err := s.SelectOne(
		&model,
		"SELECT "+precertFields+" FROM precertificates WHERE serial = ?",
		serial)
	return core.Certificate{
		RegistrationID: model.RegistrationID,
		Serial:         model.Serial,
		DER:            model.DER,
		Issued:         model.Issued,
		Expires:        model.Expires,
	}, err
}

type CertWithID struct {
	ID int64
	core.Certificate
}

// SelectCertificates selects all fields of multiple certificate objects
func SelectCertificates(s db.Selector, q string, args map[string]interface{}) ([]CertWithID, error) {
	var models []CertWithID
	_, err := s.Select(
		&models,
		"SELECT id, "+certFields+" FROM certificates "+q, args)
	return models, err
}

// SelectPrecertificates selects all fields of multiple precertificate objects.
func SelectPrecertificates(s db.Selector, q string, args map[string]interface{}) ([]CertWithID, error) {
	var models []CertWithID
	_, err := s.Select(
		&models,
		"SELECT id, "+precertFields+" FROM precertificates "+q, args)
	return models, err
}

type CertStatusMetadata struct {
	ID                    int64             `db:"id"`
	Serial                string            `db:"serial"`
	Status                core.OCSPStatus   `db:"status"`
	OCSPLastUpdated       time.Time         `db:"ocspLastUpdated"`
	RevokedDate           time.Time         `db:"revokedDate"`
	RevokedReason         revocation.Reason `db:"revokedReason"`
	LastExpirationNagSent time.Time         `db:"lastExpirationNagSent"`
	NotAfter              time.Time         `db:"notAfter"`
	IsExpired             bool              `db:"isExpired"`
	IssuerID              int64             `db:"issuerID"`
}

const certStatusFields = "id, serial, status, ocspLastUpdated, revokedDate, revokedReason, lastExpirationNagSent, ocspResponse, notAfter, isExpired, issuerID"

// SelectCertificateStatus selects all fields of one certificate status model
// identified by serial
func SelectCertificateStatus(s db.OneSelector, serial string) (core.CertificateStatus, error) {
	var model core.CertificateStatus
	err := s.SelectOne(
		&model,
		"SELECT "+certStatusFields+" FROM certificateStatus WHERE serial = ?",
		serial,
	)
	return model, err
}

// RevocationStatusModel represents a small subset of the columns in the
// certificateStatus table, used to determine the authoritative revocation
// status of a certificate.
type RevocationStatusModel struct {
	Status        core.OCSPStatus   `db:"status"`
	RevokedDate   time.Time         `db:"revokedDate"`
	RevokedReason revocation.Reason `db:"revokedReason"`
}

// SelectRevocationStatus returns the authoritative revocation information for
// the certificate with the given serial.
func SelectRevocationStatus(s db.OneSelector, serial string) (*sapb.RevocationStatus, error) {
	var model RevocationStatusModel
	err := s.SelectOne(
		&model,
		"SELECT status, revokedDate, revokedReason FROM certificateStatus WHERE serial = ?",
		serial,
	)
	if err != nil {
		return nil, err
	}

	statusInt, ok := core.OCSPStatusToInt[model.Status]
	if !ok {
		return nil, fmt.Errorf("got unrecognized status %q", model.Status)
	}

	return &sapb.RevocationStatus{
		Status:        int64(statusInt),
		RevokedDate:   timestamppb.New(model.RevokedDate),
		RevokedReason: int64(model.RevokedReason),
	}, nil
}

var mediumBlobSize = int(math.Pow(2, 24))

type issuedNameModel struct {
	ID           int64     `db:"id"`
	ReversedName string    `db:"reversedName"`
	NotBefore    time.Time `db:"notBefore"`
	Serial       string    `db:"serial"`
}

// regModel is the description of a core.Registration in the database before
type regModel struct {
	ID        int64  `db:"id"`
	Key       []byte `db:"jwk"`
	KeySHA256 string `db:"jwk_sha256"`
	Contact   string `db:"contact"`
	Agreement string `db:"agreement"`
	// InitialIP is stored as sixteen binary bytes, regardless of whether it
	// represents a v4 or v6 IP address.
	InitialIP []byte    `db:"initialIp"`
	CreatedAt time.Time `db:"createdAt"`
	LockCol   int64
	Status    string `db:"status"`
}

func registrationPbToModel(reg *corepb.Registration) (*regModel, error) {
	// Even though we don't need to convert from JSON to an in-memory JSONWebKey
	// for the sake of the `Key` field, we do need to do the conversion in order
	// to compute the SHA256 key digest.
	var jwk jose.JSONWebKey
	err := jwk.UnmarshalJSON(reg.Key)
	if err != nil {
		return nil, err
	}
	sha, err := core.KeyDigestB64(jwk.Key)
	if err != nil {
		return nil, err
	}

	// We don't want to write literal JSON "null" strings into the database if the
	// list of contact addresses is empty. Replace any possibly-`nil` slice with
	// an empty JSON array. We don't need to check reg.ContactPresent, because
	// we're going to write the whole object to the database anyway.
	jsonContact := []byte("[]")
	if len(reg.Contact) != 0 {
		jsonContact, err = json.Marshal(reg.Contact)
		if err != nil {
			return nil, err
		}
	}

	// For some reason we use different serialization formats for InitialIP
	// in database models and in protobufs, despite the fact that both formats
	// are just []byte.
	var initialIP net.IP
	err = initialIP.UnmarshalText(reg.InitialIP)
	if err != nil {
		return nil, err
	}

	// Converting the int64 zero-value to a unix timestamp does not produce
	// the time.Time zero-value (the former is 1970; the latter is year 0),
	// so we have to do this check.
	var createdAt time.Time
	if reg.CreatedAt != 0 {
		createdAt = time.Unix(0, reg.CreatedAt)
	}

	return &regModel{
		ID:        reg.Id,
		Key:       reg.Key,
		KeySHA256: sha,
		Contact:   string(jsonContact),
		Agreement: reg.Agreement,
		InitialIP: []byte(initialIP.To16()),
		CreatedAt: createdAt,
		Status:    reg.Status,
	}, nil
}

func registrationModelToPb(reg *regModel) (*corepb.Registration, error) {
	if reg.ID == 0 || len(reg.Key) == 0 || len(reg.InitialIP) == 0 {
		return nil, errors.New("incomplete Registration retrieved from DB")
	}

	contact := []string{}
	contactsPresent := false
	if len(reg.Contact) > 0 {
		err := json.Unmarshal([]byte(reg.Contact), &contact)
		if err != nil {
			return nil, err
		}
		if len(contact) > 0 {
			contactsPresent = true
		}
	}

	// For some reason we use different serialization formats for InitialIP
	// in database models and in protobufs, despite the fact that both formats
	// are just []byte.
	ipBytes, err := net.IP(reg.InitialIP).MarshalText()
	if err != nil {
		return nil, err
	}

	return &corepb.Registration{
		Id:              reg.ID,
		Key:             reg.Key,
		Contact:         contact,
		ContactsPresent: contactsPresent,
		Agreement:       reg.Agreement,
		InitialIP:       ipBytes,
		CreatedAt:       reg.CreatedAt.UTC().UnixNano(),
		Status:          reg.Status,
	}, nil
}

type recordedSerialModel struct {
	ID             int64
	Serial         string
	RegistrationID int64
	Created        time.Time
	Expires        time.Time
}

type precertificateModel struct {
	ID             int64
	Serial         string
	RegistrationID int64
	DER            []byte
	Issued         time.Time
	Expires        time.Time
}

type orderModel struct {
	ID                int64
	RegistrationID    int64
	Expires           time.Time
	Created           time.Time
	Error             []byte
	CertificateSerial string
	BeganProcessing   bool
}

type requestedNameModel struct {
	ID           int64
	OrderID      int64
	ReversedName string
}

type orderToAuthzModel struct {
	OrderID int64
	AuthzID int64
}

func orderToModel(order *corepb.Order) (*orderModel, error) {
	om := &orderModel{
		ID:                order.Id,
		RegistrationID:    order.RegistrationID,
		Expires:           time.Unix(0, order.Expires),
		Created:           time.Unix(0, order.Created),
		BeganProcessing:   order.BeganProcessing,
		CertificateSerial: order.CertificateSerial,
	}

	if order.Error != nil {
		errJSON, err := json.Marshal(order.Error)
		if err != nil {
			return nil, err
		}
		if len(errJSON) > mediumBlobSize {
			return nil, fmt.Errorf("Error object is too large to store in the database")
		}
		om.Error = errJSON
	}
	return om, nil
}

func modelToOrder(om *orderModel) (*corepb.Order, error) {
	order := &corepb.Order{
		Id:                om.ID,
		RegistrationID:    om.RegistrationID,
		Expires:           om.Expires.UnixNano(),
		Created:           om.Created.UnixNano(),
		CertificateSerial: om.CertificateSerial,
		BeganProcessing:   om.BeganProcessing,
	}
	if len(om.Error) > 0 {
		var problem corepb.ProblemDetails
		err := json.Unmarshal(om.Error, &problem)
		if err != nil {
			return &corepb.Order{}, badJSONError(
				"failed to unmarshal order model's error",
				om.Error,
				err)
		}
		order.Error = &problem
	}
	return order, nil
}

var challTypeToUint = map[string]uint8{
	"http-01":     0,
	"dns-01":      1,
	"tls-alpn-01": 2,
}

var uintToChallType = map[uint8]string{
	0: "http-01",
	1: "dns-01",
	2: "tls-alpn-01",
}

var identifierTypeToUint = map[string]uint8{
	"dns": 0,
}

var uintToIdentifierType = map[uint8]string{
	0: "dns",
}

var statusToUint = map[core.AcmeStatus]uint8{
	core.StatusPending:     0,
	core.StatusValid:       1,
	core.StatusInvalid:     2,
	core.StatusDeactivated: 3,
	core.StatusRevoked:     4,
}

var uintToStatus = map[uint8]core.AcmeStatus{
	0: core.StatusPending,
	1: core.StatusValid,
	2: core.StatusInvalid,
	3: core.StatusDeactivated,
	4: core.StatusRevoked,
}

func statusUint(status core.AcmeStatus) uint8 {
	return statusToUint[status]
}

// authzFields is used in a variety of places in sa.go, and modifications to
// it must be carried through to every use in sa.go
const authzFields = "id, identifierType, identifierValue, registrationID, status, expires, challenges, attempted, attemptedAt, token, validationError, validationRecord"

type authzModel struct {
	ID               int64      `db:"id"`
	IdentifierType   uint8      `db:"identifierType"`
	IdentifierValue  string     `db:"identifierValue"`
	RegistrationID   int64      `db:"registrationID"`
	Status           uint8      `db:"status"`
	Expires          time.Time  `db:"expires"`
	Challenges       uint8      `db:"challenges"`
	Attempted        *uint8     `db:"attempted"`
	AttemptedAt      *time.Time `db:"attemptedAt"`
	Token            []byte     `db:"token"`
	ValidationError  []byte     `db:"validationError"`
	ValidationRecord []byte     `db:"validationRecord"`
}

// SelectAuthzsMatchingIssuance looks for a set of authzs that would have
// authorized a given issuance that is known to have occurred. The returned
// authzs will all belong to the given regID, will have potentially been valid
// at the time of issuance, and will have the appropriate identifier type and
// value. This may return multiple authzs for the same identifier type and value.
//
// This returns "potentially" valid authzs because a client may have set an
// authzs status to deactivated after issuance, so we return both valid and
// deactivated authzs. It also uses a small amount of leeway (1s) to account
// for possible clock skew.
//
// This function doesn't do anything special for authzs with an expiration in
// the past. If the stored authz has a valid status, it is returned with a
// valid status regardless of whether it is also expired.
func SelectAuthzsMatchingIssuance(
	s db.Selector,
	regID int64,
	issued time.Time,
	dnsNames []string,
) ([]*corepb.Authorization, error) {
	query := fmt.Sprintf(`SELECT %s FROM authz2 WHERE
			registrationID = ? AND
			status IN (?, ?) AND
			expires >= ? AND
			attemptedAt <= ? AND
			identifierType = ? AND
			identifierValue IN (%s)`,
		authzFields,
		db.QuestionMarks(len(dnsNames)))
	var args []any
	args = append(args,
		regID,
		statusToUint[core.StatusValid],
		statusToUint[core.StatusDeactivated],
		issued.Add(-1*time.Second), // leeway for clock skew
		issued.Add(1*time.Second),  // leeway for clock skew
		identifierTypeToUint[string(identifier.DNS)],
	)
	for _, name := range dnsNames {
		args = append(args, name)
	}

	var authzModels []authzModel
	_, err := s.Select(&authzModels, query, args...)
	if err != nil {
		return nil, err
	}

	var authzs []*corepb.Authorization
	for _, model := range authzModels {
		authz, err := modelToAuthzPB(model)
		if err != nil {
			return nil, err
		}
		authzs = append(authzs, authz)

	}
	return authzs, err
}

// hasMultipleNonPendingChallenges checks if a slice of challenges contains
// more than one non-pending challenge
func hasMultipleNonPendingChallenges(challenges []*corepb.Challenge) bool {
	nonPending := false
	for _, c := range challenges {
		if c.Status == string(core.StatusValid) || c.Status == string(core.StatusInvalid) {
			if !nonPending {
				nonPending = true
			} else {
				return true
			}
		}
	}
	return false
}

// authzPBToModel converts a protobuf authorization representation to the
// authzModel storage representation.
func authzPBToModel(authz *corepb.Authorization) (*authzModel, error) {
	am := &authzModel{
		IdentifierValue: authz.Identifier,
		RegistrationID:  authz.RegistrationID,
		Status:          statusToUint[core.AcmeStatus(authz.Status)],
		Expires:         time.Unix(0, authz.Expires).UTC(),
	}
	if authz.Id != "" {
		// The v1 internal authorization objects use a string for the ID, the v2
		// storage format uses a integer ID. In order to maintain compatibility we
		// convert the integer ID to a string.
		id, err := strconv.Atoi(authz.Id)
		if err != nil {
			return nil, err
		}
		am.ID = int64(id)
	}
	if hasMultipleNonPendingChallenges(authz.Challenges) {
		return nil, errors.New("multiple challenges are non-pending")
	}
	// In the v2 authorization style we don't store individual challenges with their own
	// token, validation errors/records, etc. Instead we store a single token/error/record
	// set, a bitmap of available challenge types, and a row indicating which challenge type
	// was 'attempted'.
	//
	// Since we don't currently have the singular token/error/record set abstracted out to
	// the core authorization type yet we need to extract these from the challenges array.
	// We assume that the token in each challenge is the same and that if any of the challenges
	// has a non-pending status that it should be considered the 'attempted' challenge and
	// we extract the error/record set from that particular challenge.
	var tokenStr string
	for _, chall := range authz.Challenges {
		// Set the challenge type bit in the bitmap
		am.Challenges |= 1 << challTypeToUint[chall.Type]
		tokenStr = chall.Token
		// If the challenge status is not core.StatusPending we assume it was the 'attempted'
		// challenge and extract the relevant fields we need.
		if chall.Status == string(core.StatusValid) || chall.Status == string(core.StatusInvalid) {
			attemptedType := challTypeToUint[chall.Type]
			am.Attempted = &attemptedType

			// If validated Unix timestamp is zero then keep the core.Challenge Validated object nil.
			var validated *time.Time
			if chall.Validated != 0 {
				val := time.Unix(0, chall.Validated).UTC()
				validated = &val
			}
			am.AttemptedAt = validated

			// Marshal corepb.ValidationRecords to core.ValidationRecords so that we
			// can marshal them to JSON.
			records := make([]core.ValidationRecord, len(chall.Validationrecords))
			for i, recordPB := range chall.Validationrecords {
				var err error
				records[i], err = grpc.PBToValidationRecord(recordPB)
				if err != nil {
					return nil, err
				}
			}
			var err error
			am.ValidationRecord, err = json.Marshal(records)
			if err != nil {
				return nil, err
			}
			// If there is a error associated with the challenge marshal it to JSON
			// so that we can store it in the database.
			if chall.Error != nil {
				prob, err := grpc.PBToProblemDetails(chall.Error)
				if err != nil {
					return nil, err
				}
				am.ValidationError, err = json.Marshal(prob)
				if err != nil {
					return nil, err
				}
			}
		}
		token, err := base64.RawURLEncoding.DecodeString(tokenStr)
		if err != nil {
			return nil, err
		}
		am.Token = token
	}

	return am, nil
}

// populateAttemptedFields takes a challenge and populates it with the validation fields status,
// validation records, and error (the latter only if the validation failed) from a authzModel.
func populateAttemptedFields(am authzModel, challenge *corepb.Challenge) error {
	if len(am.ValidationError) != 0 {
		// If the error is non-empty the challenge must be invalid.
		challenge.Status = string(core.StatusInvalid)
		var prob probs.ProblemDetails
		err := json.Unmarshal(am.ValidationError, &prob)
		if err != nil {
			return badJSONError(
				"failed to unmarshal authz2 model's validation error",
				am.ValidationError,
				err)
		}
		challenge.Error, err = grpc.ProblemDetailsToPB(&prob)
		if err != nil {
			return err
		}
	} else {
		// If the error is empty the challenge must be valid.
		challenge.Status = string(core.StatusValid)
	}
	var records []core.ValidationRecord
	err := json.Unmarshal(am.ValidationRecord, &records)
	if err != nil {
		return badJSONError(
			"failed to unmarshal authz2 model's validation record",
			am.ValidationRecord,
			err)
	}
	challenge.Validationrecords = make([]*corepb.ValidationRecord, len(records))
	for i, r := range records {
		challenge.Validationrecords[i], err = grpc.ValidationRecordToPB(r)
		if err != nil {
			return err
		}
	}
	return nil
}

func modelToAuthzPB(am authzModel) (*corepb.Authorization, error) {
	pb := &corepb.Authorization{
		Id:             fmt.Sprintf("%d", am.ID),
		Status:         string(uintToStatus[am.Status]),
		Identifier:     am.IdentifierValue,
		RegistrationID: am.RegistrationID,
		Expires:        am.Expires.UTC().UnixNano(),
	}
	// Populate authorization challenge array. We do this by iterating through
	// the challenge type bitmap and creating a challenge of each type if its
	// bit is set. Each of these challenges has the token from the authorization
	// model and has its status set to core.StatusPending by default. If the
	// challenge type is equal to that in the 'attempted' row we set the status
	// to core.StatusValid or core.StatusInvalid depending on if there is anything
	// in ValidationError and populate the ValidationRecord and ValidationError
	// fields.
	for pos := uint8(0); pos < 8; pos++ {
		if (am.Challenges>>pos)&1 == 1 {
			challType := uintToChallType[pos]
			challenge := &corepb.Challenge{
				Type:   challType,
				Status: string(core.StatusPending),
				Token:  base64.RawURLEncoding.EncodeToString(am.Token),
			}
			// If the challenge type matches the attempted type it must be either
			// valid or invalid and we need to populate extra fields.
			// Also, once any challenge has been attempted, we consider the other
			// challenges "gone" per https://tools.ietf.org/html/rfc8555#section-7.1.4
			if am.Attempted != nil {
				if uintToChallType[*am.Attempted] == challType {
					err := populateAttemptedFields(am, challenge)
					if err != nil {
						return nil, err
					}
					// Get the attemptedAt time and assign to the challenge validated time.
					var validated int64
					if am.AttemptedAt != nil {
						validated = am.AttemptedAt.UTC().UnixNano()
					}
					challenge.Validated = validated
					pb.Challenges = append(pb.Challenges, challenge)
				}
			} else {
				// When no challenge has been attempted yet, all challenges are still
				// present.
				pb.Challenges = append(pb.Challenges, challenge)
			}
		}
	}
	return pb, nil
}

type keyHashModel struct {
	ID           int64
	KeyHash      []byte
	CertNotAfter time.Time
	CertSerial   string
}

var stringToSourceInt = map[string]int{
	"API":           1,
	"admin-revoker": 2,
}

// incidentModel represents a row in the 'incidents' table.
type incidentModel struct {
	ID          int64     `db:"id"`
	SerialTable string    `db:"serialTable"`
	URL         string    `db:"url"`
	RenewBy     time.Time `db:"renewBy"`
	Enabled     bool      `db:"enabled"`
}

func incidentModelToPB(i incidentModel) sapb.Incident {
	return sapb.Incident{
		Id:          i.ID,
		SerialTable: i.SerialTable,
		Url:         i.URL,
		RenewBy:     i.RenewBy.UnixNano(),
		Enabled:     i.Enabled,
	}
}

// incidentSerialModel represents a row in an 'incident_*' table.
type incidentSerialModel struct {
	Serial         string    `db:"serial"`
	RegistrationID int64     `db:"registrationID"`
	OrderID        int64     `db:"orderID"`
	LastNoticeSent time.Time `db:"lastNoticeSent"`
}

// crlEntryModel has just the certificate status fields necessary to construct
// an entry in a CRL.
type crlEntryModel struct {
	Serial        string            `db:"serial"`
	Status        core.OCSPStatus   `db:"status"`
	RevokedReason revocation.Reason `db:"revokedReason"`
	RevokedDate   time.Time         `db:"revokedDate"`
}

// HashNames returns a hash of the names requested. This is intended for use
// when interacting with the orderFqdnSets table.
func HashNames(names []string) []byte {
	names = core.UniqueLowerNames(names)
	hash := sha256.Sum256([]byte(strings.Join(names, ",")))
	return hash[:]
}

// orderFQDNSet contains the SHA256 hash of the lowercased, comma joined names
// from a new-order request, along with the corresponding orderID, the
// registration ID, and the order expiry. This is used to find
// existing orders for reuse.
type orderFQDNSet struct {
	ID             int64
	SetHash        []byte
	OrderID        int64
	RegistrationID int64
	Expires        time.Time
}

func addFQDNSet(db db.Inserter, names []string, serial string, issued time.Time, expires time.Time) error {
	return db.Insert(&core.FQDNSet{
		SetHash: HashNames(names),
		Serial:  serial,
		Issued:  issued,
		Expires: expires,
	})
}

// addOrderFQDNSet creates a new OrderFQDNSet row using the provided
// information. This function accepts a transaction so that the orderFqdnSet
// addition can take place within the order addition transaction. The caller is
// required to rollback the transaction if an error is returned.
func addOrderFQDNSet(
	db db.Inserter,
	names []string,
	orderID int64,
	regID int64,
	expires time.Time) error {
	return db.Insert(&orderFQDNSet{
		SetHash:        HashNames(names),
		OrderID:        orderID,
		RegistrationID: regID,
		Expires:        expires,
	})
}

// deleteOrderFQDNSet deletes a OrderFQDNSet row that matches the provided
// orderID. This function accepts a transaction so that the deletion can
// take place within the finalization transaction. The caller is required to
// rollback the transaction if an error is returned.
func deleteOrderFQDNSet(
	db db.Execer,
	orderID int64) error {

	result, err := db.Exec(`
	  DELETE FROM orderFqdnSets
		WHERE orderID = ?`,
		orderID)
	if err != nil {
		return err
	}
	rowsDeleted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	// We always expect there to be an order FQDN set row for each
	// pending/processing order that is being finalized. If there isn't one then
	// something is amiss and should be raised as an internal server error
	if rowsDeleted == 0 {
		return berrors.InternalServerError("No orderFQDNSet exists to delete")
	}
	return nil
}

func addIssuedNames(queryer db.Queryer, cert *x509.Certificate, isRenewal bool) error {
	if len(cert.DNSNames) == 0 {
		return berrors.InternalServerError("certificate has no DNSNames")
	}

	multiInserter, err := db.NewMultiInserter("issuedNames", []string{"reversedName", "serial", "notBefore", "renewal"}, "")
	if err != nil {
		return err
	}
	for _, name := range cert.DNSNames {
		err = multiInserter.Add([]interface{}{
			ReverseName(name),
			core.SerialToString(cert.SerialNumber),
			cert.NotBefore,
			isRenewal,
		})
		if err != nil {
			return err
		}
	}
	_, err = multiInserter.Insert(queryer)
	return err
}

func addKeyHash(db db.Inserter, cert *x509.Certificate) error {
	if cert.RawSubjectPublicKeyInfo == nil {
		return errors.New("certificate has a nil RawSubjectPublicKeyInfo")
	}
	h := sha256.Sum256(cert.RawSubjectPublicKeyInfo)
	khm := &keyHashModel{
		KeyHash:      h[:],
		CertNotAfter: cert.NotAfter,
		CertSerial:   core.SerialToString(cert.SerialNumber),
	}
	return db.Insert(khm)
}

var blockedKeysColumns = "keyHash, added, source, comment"

// statusForOrder examines the status of a provided order's authorizations to
// determine what the overall status of the order should be. In summary:
//   - If the order has an error, the order is invalid
//   - If any of the order's authorizations are in any state other than
//     valid or pending, the order is invalid.
//   - If any of the order's authorizations are pending, the order is pending.
//   - If all of the order's authorizations are valid, and there is
//     a certificate serial, the order is valid.
//   - If all of the order's authorizations are valid, and we have began
//     processing, but there is no certificate serial, the order is processing.
//   - If all of the order's authorizations are valid, and we haven't begun
//     processing, then the order is status ready.
//
// An error is returned for any other case. It assumes that the provided
// database selector already has a context associated with it.
func statusForOrder(s db.Selector, order *corepb.Order, now time.Time) (string, error) {
	// Without any further work we know an order with an error is invalid
	if order.Error != nil {
		return string(core.StatusInvalid), nil
	}

	// If the order is expired the status is invalid and we don't need to get
	// order authorizations. Its important to exit early in this case because an
	// order that references an expired authorization will be itself have been
	// expired (because we match the order expiry to the associated authz expiries
	// in ra.NewOrder), and expired authorizations may be purged from the DB.
	// Because of this purging fetching the authz's for an expired order may
	// return fewer authz objects than expected, triggering a 500 error response.
	orderExpiry := time.Unix(0, order.Expires)
	if orderExpiry.Before(now) {
		return string(core.StatusInvalid), nil
	}

	// Get the full Authorization objects for the order
	authzValidityInfo, err := getAuthorizationStatuses(s, order.V2Authorizations)
	// If there was an error getting the authorizations, return it immediately
	if err != nil {
		return "", err
	}

	// If getAuthorizationStatuses returned a different number of authorization
	// objects than the order's slice of authorization IDs something has gone
	// wrong worth raising an internal error about.
	if len(authzValidityInfo) != len(order.V2Authorizations) {
		return "", berrors.InternalServerError(
			"getAuthorizationStatuses returned the wrong number of authorization statuses "+
				"(%d vs expected %d) for order %d",
			len(authzValidityInfo), len(order.V2Authorizations), order.Id)
	}

	// Keep a count of the authorizations seen
	pendingAuthzs := 0
	validAuthzs := 0
	otherAuthzs := 0
	expiredAuthzs := 0

	// Loop over each of the order's authorization objects to examine the authz status
	for _, info := range authzValidityInfo {
		switch core.AcmeStatus(info.Status) {
		case core.StatusPending:
			pendingAuthzs++
		case core.StatusValid:
			validAuthzs++
		case core.StatusInvalid:
			otherAuthzs++
		case core.StatusDeactivated:
			otherAuthzs++
		case core.StatusRevoked:
			otherAuthzs++
		default:
			return "", berrors.InternalServerError(
				"Order is in an invalid state. Authz has invalid status %s",
				info.Status)
		}
		if info.Expires.Before(now) {
			expiredAuthzs++
		}
	}

	// An order is invalid if **any** of its authzs are invalid, deactivated,
	// revoked, or expired, see https://tools.ietf.org/html/rfc8555#section-7.1.6
	if otherAuthzs > 0 || expiredAuthzs > 0 {
		return string(core.StatusInvalid), nil
	}
	// An order is pending if **any** of its authzs are pending
	if pendingAuthzs > 0 {
		return string(core.StatusPending), nil
	}

	// An order is fully authorized if it has valid authzs for each of the order
	// names
	fullyAuthorized := len(order.Names) == validAuthzs

	// If the order isn't fully authorized we've encountered an internal error:
	// Above we checked for any invalid or pending authzs and should have returned
	// early. Somehow we made it this far but also don't have the correct number
	// of valid authzs.
	if !fullyAuthorized {
		return "", berrors.InternalServerError(
			"Order has the incorrect number of valid authorizations & no pending, " +
				"deactivated or invalid authorizations")
	}

	// If the order is fully authorized and the certificate serial is set then the
	// order is valid
	if fullyAuthorized && order.CertificateSerial != "" {
		return string(core.StatusValid), nil
	}

	// If the order is fully authorized, and we have began processing it, then the
	// order is processing.
	if fullyAuthorized && order.BeganProcessing {
		return string(core.StatusProcessing), nil
	}

	if fullyAuthorized && !order.BeganProcessing {
		return string(core.StatusReady), nil
	}

	return "", berrors.InternalServerError(
		"Order %d is in an invalid state. No state known for this order's "+
			"authorizations", order.Id)
}

type authzValidity struct {
	Status  string
	Expires time.Time
}

// getAuthorizationStatuses takes a sequence of authz IDs, and returns the
// status and expiration date of each of them. It assumes that the provided
// database selector already has a context associated with it.
func getAuthorizationStatuses(s db.Selector, ids []int64) ([]authzValidity, error) {
	var params []interface{}
	for _, id := range ids {
		params = append(params, id)
	}
	var validityInfo []struct {
		Status  uint8
		Expires time.Time
	}
	_, err := s.Select(
		&validityInfo,
		fmt.Sprintf("SELECT status, expires FROM authz2 WHERE id IN (%s)",
			db.QuestionMarks(len(ids))),
		params...,
	)
	if err != nil {
		return nil, err
	}

	allAuthzValidity := make([]authzValidity, len(validityInfo))
	for i, info := range validityInfo {
		allAuthzValidity[i] = authzValidity{
			Status:  string(uintToStatus[info.Status]),
			Expires: info.Expires,
		}
	}
	return allAuthzValidity, nil
}

// authzForOrder retrieves the authorization IDs for an order. It assumes that
// the provided database selector already has a context associated with it.
func authzForOrder(s db.Selector, orderID int64) ([]int64, error) {
	var v2IDs []int64
	_, err := s.Select(
		&v2IDs,
		"SELECT authzID FROM orderToAuthz2 WHERE orderID = ?",
		orderID,
	)
	return v2IDs, err
}

// namesForOrder finds all of the requested names associated with an order. The
// names are returned in their reversed form (see `sa.ReverseName`). It assumes
// that the provided database selector already has a context associated with it.
func namesForOrder(s db.Selector, orderID int64) ([]string, error) {
	var reversedNames []string
	_, err := s.Select(
		&reversedNames,
		`SELECT reversedName
	   FROM requestedNames
	   WHERE orderID = ?`,
		orderID)
	if err != nil {
		return nil, err
	}
	return reversedNames, nil
}
