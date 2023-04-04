package sa

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmhodges/clock"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ocsp"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/letsencrypt/boulder/core"
	corepb "github.com/letsencrypt/boulder/core/proto"
	"github.com/letsencrypt/boulder/db"
	berrors "github.com/letsencrypt/boulder/errors"
	"github.com/letsencrypt/boulder/features"
	bgrpc "github.com/letsencrypt/boulder/grpc"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/revocation"
	sapb "github.com/letsencrypt/boulder/sa/proto"
)

var (
	errIncompleteRequest = errors.New("incomplete gRPC request message")
)

// SQLStorageAuthority defines a Storage Authority.
//
// Note that although SQLStorageAuthority does have methods wrapping all of the
// read-only methods provided by the SQLStorageAuthorityRO, those wrapper
// implementations are in saro.go, next to the real implementations.
type SQLStorageAuthority struct {
	sapb.UnimplementedStorageAuthorityServer
	*SQLStorageAuthorityRO

	dbMap *db.WrappedMap

	// rateLimitWriteErrors is a Counter for the number of times
	// a ratelimit update transaction failed during AddCertificate request
	// processing. We do not fail the overall AddCertificate call when ratelimit
	// transactions fail and so use this stat to maintain visibility into the rate
	// this occurs.
	rateLimitWriteErrors prometheus.Counter
}

// NewSQLStorageAuthorityWrapping provides persistence using a SQL backend for
// Boulder. It takes a read-only storage authority to wrap, which is useful if
// you are constructing both types of implementations and want to share
// read-only database connections between them.
func NewSQLStorageAuthorityWrapping(
	ssaro *SQLStorageAuthorityRO,
	dbMap *db.WrappedMap,
	stats prometheus.Registerer,
) (*SQLStorageAuthority, error) {
	SetSQLDebug(dbMap, ssaro.log)

	rateLimitWriteErrors := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "rate_limit_write_errors",
		Help: "number of failed ratelimit update transactions during AddCertificate",
	})
	stats.MustRegister(rateLimitWriteErrors)

	ssa := &SQLStorageAuthority{
		SQLStorageAuthorityRO: ssaro,
		dbMap:                 dbMap,
		rateLimitWriteErrors:  rateLimitWriteErrors,
	}

	return ssa, nil
}

// NewSQLStorageAuthority provides persistence using a SQL backend for
// Boulder. It constructs its own read-only storage authority to wrap.
func NewSQLStorageAuthority(
	dbMap *db.WrappedMap,
	dbReadOnlyMap *db.WrappedMap,
	dbIncidentsMap *db.WrappedMap,
	parallelismPerRPC int,
	clk clock.Clock,
	logger blog.Logger,
	stats prometheus.Registerer,
) (*SQLStorageAuthority, error) {
	ssaro, err := NewSQLStorageAuthorityRO(dbReadOnlyMap, dbIncidentsMap, parallelismPerRPC, clk, logger)
	if err != nil {
		return nil, err
	}

	return NewSQLStorageAuthorityWrapping(ssaro, dbMap, stats)
}

// NewRegistration stores a new Registration
func (ssa *SQLStorageAuthority) NewRegistration(ctx context.Context, req *corepb.Registration) (*corepb.Registration, error) {
	if len(req.Key) == 0 || len(req.InitialIP) == 0 {
		return nil, errIncompleteRequest
	}

	reg, err := registrationPbToModel(req)
	if err != nil {
		return nil, err
	}

	reg.CreatedAt = ssa.clk.Now()

	err = ssa.dbMap.WithContext(ctx).Insert(reg)
	if err != nil {
		if db.IsDuplicate(err) {
			// duplicate entry error can only happen when jwk_sha256 collides, indicate
			// to caller that the provided key is already in use
			return nil, berrors.DuplicateError("key is already in use for a different account")
		}
		return nil, err
	}
	return registrationModelToPb(reg)
}

// UpdateRegistration stores an updated Registration
func (ssa *SQLStorageAuthority) UpdateRegistration(ctx context.Context, req *corepb.Registration) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 || len(req.Key) == 0 || len(req.InitialIP) == 0 {
		return nil, errIncompleteRequest
	}

	const query = "WHERE id = ?"
	curr, err := selectRegistration(ssa.dbMap.WithContext(ctx), query, req.Id)
	if err != nil {
		if db.IsNoRows(err) {
			return nil, berrors.NotFoundError("registration with ID '%d' not found", req.Id)
		}
		return nil, err
	}

	update, err := registrationPbToModel(req)
	if err != nil {
		return nil, err
	}

	// Copy the existing registration model's LockCol to the new updated
	// registration model's LockCol
	update.LockCol = curr.LockCol
	n, err := ssa.dbMap.WithContext(ctx).Update(update)
	if err != nil {
		if db.IsDuplicate(err) {
			// duplicate entry error can only happen when jwk_sha256 collides, indicate
			// to caller that the provided key is already in use
			return nil, berrors.DuplicateError("key is already in use for a different account")
		}
		return nil, err
	}
	if n == 0 {
		return nil, berrors.NotFoundError("registration with ID '%d' not found", req.Id)
	}

	return &emptypb.Empty{}, nil
}

// AddCertificate stores an issued certificate, returning an error if it is a
// duplicate or if any other failure occurs.
func (ssa *SQLStorageAuthority) AddCertificate(ctx context.Context, req *sapb.AddCertificateRequest) (*emptypb.Empty, error) {
	if len(req.Der) == 0 || req.RegID == 0 || req.Issued == 0 {
		return nil, errIncompleteRequest
	}
	parsedCertificate, err := x509.ParseCertificate(req.Der)
	if err != nil {
		return nil, err
	}
	digest := core.Fingerprint256(req.Der)
	serial := core.SerialToString(parsedCertificate.SerialNumber)

	cert := &core.Certificate{
		RegistrationID: req.RegID,
		Serial:         serial,
		Digest:         digest,
		DER:            req.Der,
		Issued:         time.Unix(0, req.Issued),
		Expires:        parsedCertificate.NotAfter,
	}

	isRenewalRaw, overallError := db.WithTransaction(ctx, ssa.dbMap, func(txWithCtx db.Executor) (interface{}, error) {
		// Select to see if cert exists
		var row struct {
			Count int64
		}
		err := txWithCtx.SelectOne(&row, "SELECT COUNT(*) as count FROM certificates WHERE serial=?", serial)
		if err != nil {
			return nil, err
		}
		if row.Count > 0 {
			return nil, berrors.DuplicateError("cannot add a duplicate cert")
		}

		// Save the final certificate
		err = txWithCtx.Insert(cert)
		if err != nil {
			return nil, err
		}

		// NOTE(@cpu): When we collect up names to check if an FQDN set exists (e.g.
		// that it is a renewal) we use just the DNSNames from the certificate and
		// ignore the Subject Common Name (if any). This is a safe assumption because
		// if a certificate we issued were to have a Subj. CN not present as a SAN it
		// would be a misissuance and miscalculating whether the cert is a renewal or
		// not for the purpose of rate limiting is the least of our troubles.
		isRenewal, err := ssa.checkFQDNSetExists(
			txWithCtx.SelectOne,
			parsedCertificate.DNSNames)
		if err != nil {
			return nil, err
		}

		return isRenewal, err
	})
	if overallError != nil {
		return nil, overallError
	}

	// Recast the interface{} return from db.WithTransaction as a bool, returning
	// an error if we can't.
	var isRenewal bool
	if boolVal, ok := isRenewalRaw.(bool); !ok {
		return nil, fmt.Errorf(
			"AddCertificate db.WithTransaction returned %T out var, expected bool",
			isRenewalRaw)
	} else {
		isRenewal = boolVal
	}

	// In a separate transaction perform the work required to update tables used
	// for rate limits. Since the effects of failing these writes is slight
	// miscalculation of rate limits we choose to not fail the AddCertificate
	// operation if the rate limit update transaction fails.
	_, rlTransactionErr := db.WithTransaction(ctx, ssa.dbMap, func(txWithCtx db.Executor) (interface{}, error) {
		// Add to the rate limit table, but only for new certificates. Renewals
		// don't count against the certificatesPerName limit.
		if !isRenewal {
			timeToTheHour := parsedCertificate.NotBefore.Round(time.Hour)
			err := ssa.addCertificatesPerName(txWithCtx, parsedCertificate.DNSNames, timeToTheHour)
			if err != nil {
				return nil, err
			}
		}

		// Update the FQDN sets now that there is a final certificate to ensure rate
		// limits are calculated correctly.
		err = addFQDNSet(
			txWithCtx,
			parsedCertificate.DNSNames,
			core.SerialToString(parsedCertificate.SerialNumber),
			parsedCertificate.NotBefore,
			parsedCertificate.NotAfter,
		)
		if err != nil {
			return nil, err
		}

		return nil, nil
	})
	// If the ratelimit transaction failed increment a stat and log a warning
	// but don't return an error from AddCertificate.
	if rlTransactionErr != nil {
		ssa.rateLimitWriteErrors.Inc()
		ssa.log.AuditErrf("failed AddCertificate ratelimit update transaction: %v", rlTransactionErr)
	}

	return &emptypb.Empty{}, nil
}

// DeactivateRegistration deactivates a currently valid registration
func (ssa *SQLStorageAuthority) DeactivateRegistration(ctx context.Context, req *sapb.RegistrationID) (*emptypb.Empty, error) {
	if req == nil || req.Id == 0 {
		return nil, errIncompleteRequest
	}
	_, err := ssa.dbMap.WithContext(ctx).Exec(
		"UPDATE registrations SET status = ? WHERE status = ? AND id = ?",
		string(core.StatusDeactivated),
		string(core.StatusValid),
		req.Id,
	)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// DeactivateAuthorization2 deactivates a currently valid or pending authorization.
func (ssa *SQLStorageAuthority) DeactivateAuthorization2(ctx context.Context, req *sapb.AuthorizationID2) (*emptypb.Empty, error) {
	if req.Id == 0 {
		return nil, errIncompleteRequest
	}

	_, err := ssa.dbMap.Exec(
		`UPDATE authz2 SET status = :deactivated WHERE id = :id and status IN (:valid,:pending)`,
		map[string]interface{}{
			"deactivated": statusUint(core.StatusDeactivated),
			"id":          req.Id,
			"valid":       statusUint(core.StatusValid),
			"pending":     statusUint(core.StatusPending),
		},
	)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// NewOrderAndAuthzs adds the given authorizations to the database, adds their
// autogenerated IDs to the given order, and then adds the order to the db.
// This is done inside a single transaction to prevent situations where new
// authorizations are created, but then their corresponding order is never
// created, leading to "invisible" pending authorizations.
func (ssa *SQLStorageAuthority) NewOrderAndAuthzs(ctx context.Context, req *sapb.NewOrderAndAuthzsRequest) (*corepb.Order, error) {
	if req.NewOrder == nil {
		return nil, errIncompleteRequest
	}

	output, err := db.WithTransaction(ctx, ssa.dbMap, func(txWithCtx db.Executor) (interface{}, error) {
		// First, insert all of the new authorizations and record their IDs.
		newAuthzIDs := make([]int64, 0)
		if len(req.NewAuthzs) != 0 {
			inserter, err := db.NewMultiInserter("authz2", strings.Split(authzFields, ", "), "id")
			if err != nil {
				return nil, err
			}
			for _, authz := range req.NewAuthzs {
				if authz.Status != string(core.StatusPending) {
					return nil, berrors.InternalServerError("authorization must be pending")
				}
				am, err := authzPBToModel(authz)
				if err != nil {
					return nil, err
				}
				err = inserter.Add([]interface{}{
					am.ID,
					am.IdentifierType,
					am.IdentifierValue,
					am.RegistrationID,
					statusToUint[core.StatusPending],
					am.Expires,
					am.Challenges,
					nil,
					nil,
					am.Token,
					nil,
					nil,
				})
				if err != nil {
					return nil, err
				}
			}
			newAuthzIDs, err = inserter.Insert(txWithCtx)
			if err != nil {
				return nil, err
			}
		}

		// Second, insert the new order.
		order := &orderModel{
			RegistrationID: req.NewOrder.RegistrationID,
			Expires:        time.Unix(0, req.NewOrder.Expires),
			Created:        ssa.clk.Now(),
		}
		err := txWithCtx.Insert(order)
		if err != nil {
			return nil, err
		}

		// Third, insert all of the orderToAuthz relations.
		inserter, err := db.NewMultiInserter("orderToAuthz2", []string{"orderID", "authzID"}, "")
		if err != nil {
			return nil, err
		}
		for _, id := range req.NewOrder.V2Authorizations {
			err = inserter.Add([]interface{}{order.ID, id})
			if err != nil {
				return nil, err
			}
		}
		for _, id := range newAuthzIDs {
			err = inserter.Add([]interface{}{order.ID, id})
			if err != nil {
				return nil, err
			}
		}
		_, err = inserter.Insert(txWithCtx)
		if err != nil {
			return nil, err
		}

		// Fourth, insert all of the requestedNames.
		inserter, err = db.NewMultiInserter("requestedNames", []string{"orderID", "reversedName"}, "")
		if err != nil {
			return nil, err
		}
		for _, name := range req.NewOrder.Names {
			err = inserter.Add([]interface{}{order.ID, ReverseName(name)})
			if err != nil {
				return nil, err
			}
		}
		_, err = inserter.Insert(txWithCtx)
		if err != nil {
			return nil, err
		}

		// Fifth, insert the FQDNSet entry for the order.
		err = addOrderFQDNSet(txWithCtx, req.NewOrder.Names, order.ID, order.RegistrationID, order.Expires)
		if err != nil {
			return nil, err
		}

		// Finally, build the overall Order PB.
		res := &corepb.Order{
			// ID and Created were auto-populated on the order model when it was inserted.
			Id:      order.ID,
			Created: order.Created.UnixNano(),
			// These are carried over from the original request unchanged.
			RegistrationID: req.NewOrder.RegistrationID,
			Expires:        req.NewOrder.Expires,
			Names:          req.NewOrder.Names,
			// Have to combine the already-associated and newly-reacted authzs.
			V2Authorizations: append(req.NewOrder.V2Authorizations, newAuthzIDs...),
			// A new order is never processing because it can't be finalized yet.
			BeganProcessing: false,
		}

		// Calculate the order status before returning it. Since it may have reused
		// all valid authorizations the order may be "born" in a ready status.
		status, err := statusForOrder(txWithCtx, res, ssa.clk.Now())
		if err != nil {
			return nil, err
		}
		res.Status = status

		return res, nil
	})
	if err != nil {
		return nil, err
	}

	order, ok := output.(*corepb.Order)
	if !ok {
		return nil, fmt.Errorf("casting error in NewOrderAndAuthzs")
	}

	// Increment the order creation count
	err = addNewOrdersRateLimit(ssa.dbMap.WithContext(ctx), req.NewOrder.RegistrationID, ssa.clk.Now().Truncate(time.Minute))
	if err != nil {
		return nil, err
	}

	return order, nil
}

// SetOrderProcessing updates an order from pending status to processing
// status by updating the `beganProcessing` field of the corresponding
// Order table row in the DB.
func (ssa *SQLStorageAuthority) SetOrderProcessing(ctx context.Context, req *sapb.OrderRequest) (*emptypb.Empty, error) {
	if req.Id == 0 {
		return nil, errIncompleteRequest
	}
	_, overallError := db.WithTransaction(ctx, ssa.dbMap, func(txWithCtx db.Executor) (interface{}, error) {
		result, err := txWithCtx.Exec(`
		UPDATE orders
		SET beganProcessing = ?
		WHERE id = ?
		AND beganProcessing = ?`,
			true,
			req.Id,
			false)
		if err != nil {
			return nil, berrors.InternalServerError("error updating order to beganProcessing status")
		}

		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return nil, berrors.OrderNotReadyError("Order was already processing. This may indicate your client finalized the same order multiple times, possibly due to a client bug.")
		}

		return nil, nil
	})
	if overallError != nil {
		return nil, overallError
	}
	return &emptypb.Empty{}, nil
}

// SetOrderError updates a provided Order's error field.
func (ssa *SQLStorageAuthority) SetOrderError(ctx context.Context, req *sapb.SetOrderErrorRequest) (*emptypb.Empty, error) {
	if req.Id == 0 || req.Error == nil {
		return nil, errIncompleteRequest
	}
	_, overallError := db.WithTransaction(ctx, ssa.dbMap, func(txWithCtx db.Executor) (interface{}, error) {
		om, err := orderToModel(&corepb.Order{
			Id:    req.Id,
			Error: req.Error,
		})
		if err != nil {
			return nil, err
		}

		result, err := txWithCtx.Exec(`
		UPDATE orders
		SET error = ?
		WHERE id = ?`,
			om.Error,
			om.ID)
		if err != nil {
			return nil, berrors.InternalServerError("error updating order error field")
		}

		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return nil, berrors.InternalServerError("no order updated with new error field")
		}

		return nil, nil
	})
	if overallError != nil {
		return nil, overallError
	}
	return &emptypb.Empty{}, nil
}

// FinalizeOrder finalizes a provided *corepb.Order by persisting the
// CertificateSerial and a valid status to the database. No fields other than
// CertificateSerial and the order ID on the provided order are processed (e.g.
// this is not a generic update RPC).
func (ssa *SQLStorageAuthority) FinalizeOrder(ctx context.Context, req *sapb.FinalizeOrderRequest) (*emptypb.Empty, error) {
	if req.Id == 0 || req.CertificateSerial == "" {
		return nil, errIncompleteRequest
	}
	_, overallError := db.WithTransaction(ctx, ssa.dbMap, func(txWithCtx db.Executor) (interface{}, error) {
		result, err := txWithCtx.Exec(`
		UPDATE orders
		SET certificateSerial = ?
		WHERE id = ? AND
		beganProcessing = true`,
			req.CertificateSerial,
			req.Id)
		if err != nil {
			return nil, berrors.InternalServerError("error updating order for finalization")
		}

		n, err := result.RowsAffected()
		if err != nil || n == 0 {
			return nil, berrors.InternalServerError("no order updated for finalization")
		}

		// Delete the orderFQDNSet row for the order now that it has been finalized.
		// We use this table for order reuse and should not reuse a finalized order.
		err = deleteOrderFQDNSet(txWithCtx, req.Id)
		if err != nil {
			return nil, err
		}

		return nil, nil
	})
	if overallError != nil {
		return nil, overallError
	}
	return &emptypb.Empty{}, nil
}

// FinalizeAuthorization2 moves a pending authorization to either the valid or invalid status. If
// the authorization is being moved to invalid the validationError field must be set. If the
// authorization is being moved to valid the validationRecord and expires fields must be set.
func (ssa *SQLStorageAuthority) FinalizeAuthorization2(ctx context.Context, req *sapb.FinalizeAuthorizationRequest) (*emptypb.Empty, error) {
	if req.Status == "" || req.Attempted == "" || req.Expires == 0 || req.Id == 0 {
		return nil, errIncompleteRequest
	}

	if req.Status != string(core.StatusValid) && req.Status != string(core.StatusInvalid) {
		return nil, berrors.InternalServerError("authorization must have status valid or invalid")
	}
	query := `UPDATE authz2 SET
		status = :status,
		attempted = :attempted,
		attemptedAt = :attemptedAt,
		validationRecord = :validationRecord,
		validationError = :validationError,
		expires = :expires
		WHERE id = :id AND status = :pending`
	var validationRecords []core.ValidationRecord
	for _, recordPB := range req.ValidationRecords {
		record, err := bgrpc.PBToValidationRecord(recordPB)
		if err != nil {
			return nil, err
		}
		validationRecords = append(validationRecords, record)
	}
	vrJSON, err := json.Marshal(validationRecords)
	if err != nil {
		return nil, err
	}
	var veJSON []byte
	if req.ValidationError != nil {
		validationError, err := bgrpc.PBToProblemDetails(req.ValidationError)
		if err != nil {
			return nil, err
		}
		j, err := json.Marshal(validationError)
		if err != nil {
			return nil, err
		}
		veJSON = j
	}
	// Check to see if the AttemptedAt time is non zero and convert to
	// *time.Time if so. If it is zero, leave nil and don't convert. Keep
	// the the database attemptedAt field Null instead of
	// 1970-01-01 00:00:00.
	var attemptedTime *time.Time
	if req.AttemptedAt != 0 {
		val := time.Unix(0, req.AttemptedAt).UTC()
		attemptedTime = &val
	}
	params := map[string]interface{}{
		"status":           statusToUint[core.AcmeStatus(req.Status)],
		"attempted":        challTypeToUint[req.Attempted],
		"attemptedAt":      attemptedTime,
		"validationRecord": vrJSON,
		"id":               req.Id,
		"pending":          statusUint(core.StatusPending),
		"expires":          time.Unix(0, req.Expires).UTC(),
		// if req.ValidationError is nil veJSON should also be nil
		// which should result in a NULL field
		"validationError": veJSON,
	}

	res, err := ssa.dbMap.Exec(query, params)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, berrors.NotFoundError("authorization with id %d not found", req.Id)
	} else if rows > 1 {
		return nil, berrors.InternalServerError("multiple rows updated for authorization id %d", req.Id)
	}
	return &emptypb.Empty{}, nil
}

// RevokeCertificate stores revocation information about a certificate. It will only store this
// information if the certificate is not already marked as revoked.
func (ssa *SQLStorageAuthority) RevokeCertificate(ctx context.Context, req *sapb.RevokeCertificateRequest) (*emptypb.Empty, error) {
	if req.Serial == "" || req.Date == 0 {
		return nil, errIncompleteRequest
	}
	if req.Response == nil && !features.Enabled(features.ROCSPStage6) {
		return nil, errIncompleteRequest
	}

	revokedDate := time.Unix(0, req.Date)
	ocspResponse := req.Response
	if features.Enabled(features.ROCSPStage6) {
		ocspResponse = nil
	}

	res, err := ssa.dbMap.Exec(
		`UPDATE certificateStatus SET
				status = ?,
				revokedReason = ?,
				revokedDate = ?,
				ocspLastUpdated = ?,
				ocspResponse = ?
			WHERE serial = ? AND status != ?`,
		string(core.OCSPStatusRevoked),
		revocation.Reason(req.Reason),
		revokedDate,
		revokedDate,
		ocspResponse,
		req.Serial,
		string(core.OCSPStatusRevoked),
	)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		return nil, berrors.AlreadyRevokedError("no certificate with serial %s and status other than %s", req.Serial, string(core.OCSPStatusRevoked))
	}

	return &emptypb.Empty{}, nil
}

// UpdateRevokedCertificate stores new revocation information about an
// already-revoked certificate. It will only store this information if the
// cert is already revoked, if the new revocation reason is `KeyCompromise`,
// and if the revokedDate is identical to the current revokedDate.
func (ssa *SQLStorageAuthority) UpdateRevokedCertificate(ctx context.Context, req *sapb.RevokeCertificateRequest) (*emptypb.Empty, error) {
	if req.Serial == "" || req.Date == 0 || req.Backdate == 0 {
		return nil, errIncompleteRequest
	}
	if req.Response == nil && !features.Enabled(features.ROCSPStage6) {
		return nil, errIncompleteRequest
	}
	if req.Reason != ocsp.KeyCompromise {
		return nil, fmt.Errorf("cannot update revocation for any reason other than keyCompromise (1); got: %d", req.Reason)
	}

	thisUpdate := time.Unix(0, req.Date)
	revokedDate := time.Unix(0, req.Backdate)
	ocspResponse := req.Response
	if features.Enabled(features.ROCSPStage6) {
		ocspResponse = nil
	}

	res, err := ssa.dbMap.Exec(
		`UPDATE certificateStatus SET
				revokedReason = ?,
				ocspLastUpdated = ?,
				ocspResponse = ?
			WHERE serial = ? AND status = ? AND revokedReason != ? AND revokedDate = ?`,
		revocation.Reason(ocsp.KeyCompromise),
		thisUpdate,
		ocspResponse,
		req.Serial,
		string(core.OCSPStatusRevoked),
		revocation.Reason(ocsp.KeyCompromise),
		revokedDate,
	)
	if err != nil {
		return nil, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if rows == 0 {
		// InternalServerError because we expected this certificate status to exist,
		// to already be revoked for a different reason, and to have a matching date.
		return nil, berrors.InternalServerError("no certificate with serial %s and revoked reason other than keyCompromise", req.Serial)
	}

	return &emptypb.Empty{}, nil
}

// AddBlockedKey adds a key hash to the blockedKeys table
func (ssa *SQLStorageAuthority) AddBlockedKey(ctx context.Context, req *sapb.AddBlockedKeyRequest) (*emptypb.Empty, error) {
	if core.IsAnyNilOrZero(req.KeyHash, req.Added, req.Source) {
		return nil, errIncompleteRequest
	}
	sourceInt, ok := stringToSourceInt[req.Source]
	if !ok {
		return nil, errors.New("unknown source")
	}
	cols, qs := blockedKeysColumns, "?, ?, ?, ?"
	vals := []interface{}{
		req.KeyHash,
		time.Unix(0, req.Added),
		sourceInt,
		req.Comment,
	}
	if req.RevokedBy != 0 {
		cols += ", revokedBy"
		qs += ", ?"
		vals = append(vals, req.RevokedBy)
	}
	_, err := ssa.dbMap.Exec(
		fmt.Sprintf("INSERT INTO blockedKeys (%s) VALUES (%s)", cols, qs),
		vals...,
	)
	if err != nil {
		if db.IsDuplicate(err) {
			// Ignore duplicate inserts so multiple certs with the same key can
			// be revoked.
			return &emptypb.Empty{}, nil
		}
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
