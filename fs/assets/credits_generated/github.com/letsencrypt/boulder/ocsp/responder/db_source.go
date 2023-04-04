package responder

import (
	"context"
	"fmt"

	"github.com/go-gorp/gorp/v3"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/db"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/sa"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/ocsp"
)

type dbSource struct {
	dbMap   dbSelector
	counter *prometheus.CounterVec
	log     blog.Logger
}

// dbSelector is a limited subset of the db.WrappedMap interface to allow for
// easier mocking of mysql operations in tests.
type dbSelector interface {
	SelectOne(holder interface{}, query string, args ...interface{}) error
	WithContext(ctx context.Context) gorp.SqlExecutor
}

// NewDbSource returns a dbSource which will look up OCSP responses in a SQL
// database.
func NewDbSource(dbMap dbSelector, stats prometheus.Registerer, log blog.Logger) (*dbSource, error) {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "ocsp_db_responses",
		Help: "Count of OCSP requests/responses by action taken by the dbSource",
	}, []string{"result"})
	stats.MustRegister(counter)

	return &dbSource{
		dbMap:   dbMap,
		counter: counter,
		log:     log,
	}, nil
}

// Response implements the Source interface. It looks up the requested OCSP
// response in the sql database. If the certificate status row that it finds
// indicates that the cert is expired or this cert has never had an OCSP
// response generated for it, it returns an error.
func (src *dbSource) Response(ctx context.Context, req *ocsp.Request) (*Response, error) {
	serialString := core.SerialToString(req.SerialNumber)

	certStatus, err := sa.SelectCertificateStatus(src.dbMap.WithContext(ctx), serialString)
	if err != nil {
		if db.IsNoRows(err) {
			src.counter.WithLabelValues("not_found").Inc()
			return nil, ErrNotFound
		}

		src.log.AuditErrf("Looking up OCSP response in DB: %s", err)
		src.counter.WithLabelValues("lookup_error").Inc()
		return nil, err
	}

	if certStatus.IsExpired {
		src.counter.WithLabelValues("expired").Inc()
		return nil, fmt.Errorf("certificate is expired: %w", ErrNotFound)
	} else if certStatus.OCSPLastUpdated.IsZero() {
		src.counter.WithLabelValues("never_updated").Inc()
		return nil, fmt.Errorf("certificate has a zero OCSPLastUpdated: %w", ErrNotFound)
	}

	resp, err := ocsp.ParseResponse(certStatus.OCSPResponse, nil)
	if err != nil {
		src.counter.WithLabelValues("parse_error").Inc()
		return nil, err
	}

	src.counter.WithLabelValues("success").Inc()
	return &Response{Response: resp, Raw: certStatus.OCSPResponse}, nil
}
