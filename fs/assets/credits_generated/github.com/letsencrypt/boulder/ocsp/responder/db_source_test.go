package responder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/go-gorp/gorp/v3"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/db"
	blog "github.com/letsencrypt/boulder/log"
	"github.com/letsencrypt/boulder/metrics"
	"github.com/letsencrypt/boulder/test"
	"golang.org/x/crypto/ocsp"
)

// echoSelector always returns the given certificateStatus.
type echoSelector struct {
	db.MockSqlExecutor
	status core.CertificateStatus
}

func (s echoSelector) WithContext(context.Context) gorp.SqlExecutor {
	return s
}

func (s echoSelector) SelectOne(output interface{}, _ string, _ ...interface{}) error {
	outputPtr, ok := output.(*core.CertificateStatus)
	if !ok {
		return fmt.Errorf("incorrect output type %T", output)
	}
	*outputPtr = s.status
	return nil
}

// errorSelector always returns the given error.
type errorSelector struct {
	db.MockSqlExecutor
	err error
}

func (s errorSelector) SelectOne(_ interface{}, _ string, _ ...interface{}) error {
	return s.err
}

func (s errorSelector) WithContext(context.Context) gorp.SqlExecutor {
	return s
}

func TestDbSource(t *testing.T) {
	reqBytes, err := os.ReadFile("./testdata/ocsp.req")
	test.AssertNotError(t, err, "failed to read OCSP request")
	req, err := ocsp.ParseRequest(reqBytes)
	test.AssertNotError(t, err, "failed to parse OCSP request")

	respBytes, err := os.ReadFile("./testdata/ocsp.resp")
	test.AssertNotError(t, err, "failed to read OCSP response")

	// Test for failure when the database lookup fails.
	dbErr := errors.New("something went wrong")
	src, err := NewDbSource(errorSelector{err: dbErr}, metrics.NoopRegisterer, blog.NewMock())
	test.AssertNotError(t, err, "failed to create dbSource")
	_, err = src.Response(context.Background(), req)
	test.AssertEquals(t, err, dbErr)

	// Test for graceful recovery when the database returns no results.
	dbErr = db.ErrDatabaseOp{
		Op:    "test",
		Table: "certificateStatus",
		Err:   sql.ErrNoRows,
	}
	src, err = NewDbSource(errorSelector{err: dbErr}, metrics.NoopRegisterer, blog.NewMock())
	test.AssertNotError(t, err, "failed to create dbSource")
	_, err = src.Response(context.Background(), req)
	test.AssertErrorIs(t, err, ErrNotFound)

	// Test for converting expired results into no results.
	status := core.CertificateStatus{
		IsExpired: true,
	}
	src, err = NewDbSource(echoSelector{status: status}, metrics.NoopRegisterer, blog.NewMock())
	test.AssertNotError(t, err, "failed to create dbSource")
	_, err = src.Response(context.Background(), req)
	test.AssertErrorIs(t, err, ErrNotFound)

	// Test for converting never-updated results into no results.
	status = core.CertificateStatus{
		IsExpired:       false,
		OCSPLastUpdated: time.Time{},
	}
	src, err = NewDbSource(echoSelector{status: status}, metrics.NoopRegisterer, blog.NewMock())
	test.AssertNotError(t, err, "failed to create dbSource")
	_, err = src.Response(context.Background(), req)
	test.AssertErrorIs(t, err, ErrNotFound)

	// Test for reporting parse errors.
	status = core.CertificateStatus{
		IsExpired:       false,
		OCSPLastUpdated: time.Now(),
		OCSPResponse:    respBytes[1:],
	}
	src, err = NewDbSource(echoSelector{status: status}, metrics.NoopRegisterer, blog.NewMock())
	test.AssertNotError(t, err, "failed to create dbSource")
	_, err = src.Response(context.Background(), req)
	test.AssertError(t, err, "expected failure")

	// Test the happy path.
	status = core.CertificateStatus{
		IsExpired:       false,
		OCSPLastUpdated: time.Now(),
		OCSPResponse:    respBytes,
	}
	src, err = NewDbSource(echoSelector{status: status}, metrics.NoopRegisterer, blog.NewMock())
	test.AssertNotError(t, err, "failed to create dbSource")
	_, err = src.Response(context.Background(), req)
	test.AssertNotError(t, err, "unexpected failure")
}
