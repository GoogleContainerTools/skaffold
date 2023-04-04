package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	gorp "github.com/go-gorp/gorp/v3"

	"github.com/go-sql-driver/mysql"
	"github.com/letsencrypt/boulder/core"
	"github.com/letsencrypt/boulder/test"
	"github.com/letsencrypt/boulder/test/vars"
)

func TestErrDatabaseOpError(t *testing.T) {
	testErr := errors.New("computers are cancelled")
	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name: "error with table",
			err: ErrDatabaseOp{
				Op:    "test",
				Table: "testTable",
				Err:   testErr,
			},
			expected: fmt.Sprintf("failed to test testTable: %s", testErr),
		},
		{
			name: "error with no table",
			err: ErrDatabaseOp{
				Op:  "test",
				Err: testErr,
			},
			expected: fmt.Sprintf("failed to test: %s", testErr),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test.AssertEquals(t, tc.err.Error(), tc.expected)
		})
	}
}

func TestErrDatabaseOpNoRows(t *testing.T) {
	testCases := []struct {
		name           string
		err            ErrDatabaseOp
		expectedNoRows bool
	}{
		{
			name: "underlying err is sql.ErrNoRows",
			err: ErrDatabaseOp{
				Op:    "test",
				Table: "testTable",
				Err:   sql.ErrNoRows,
			},
			expectedNoRows: true,
		},
		{
			name: "underlying err is not sql.ErrNoRows",
			err: ErrDatabaseOp{
				Op:    "test",
				Table: "testTable",
				Err:   errors.New("lots of rows. too many rows."),
			},
			expectedNoRows: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test.AssertEquals(t, tc.err.noRows(), tc.expectedNoRows)
		})
	}
}

func TestErrDatabaseOpDuplicate(t *testing.T) {
	testCases := []struct {
		name            string
		err             ErrDatabaseOp
		expectDuplicate bool
	}{
		{
			name: "underlying err has duplicate prefix",
			err: ErrDatabaseOp{
				Op:    "test",
				Table: "testTable",
				Err:   errors.New("Error 1062: Duplicate entry detected!!!!!!!"),
			},
			expectDuplicate: true,
		},
		{
			name: "underlying err doesn't have duplicate prefix",
			err: ErrDatabaseOp{
				Op:    "test",
				Table: "testTable",
				Err:   errors.New("DB forgot to save your data."),
			},
			expectDuplicate: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			test.AssertEquals(t, tc.err.duplicate(), tc.expectDuplicate)
		})
	}
}

func TestTableFromQuery(t *testing.T) {
	// A sample of example queries logged by the SA during Boulder
	// unit/integration tests.
	testCases := []struct {
		query         string
		expectedTable string
	}{
		{
			query:         "SELECT id, jwk, jwk_sha256, contact, agreement, initialIP, createdAt, LockCol, status FROM registrations WHERE jwk_sha256 = ?",
			expectedTable: "registrations",
		},
		{
			query:         "\n\t\t\t\t\tSELECT orderID, registrationID\n\t\t\t\t\tFROM orderFqdnSets\n\t\t\t\t\tWHERE setHash = ?\n\t\t\t\t\tAND expires > ?\n\t\t\t\t\tORDER BY expires ASC\n\t\t\t\t\tLIMIT 1",
			expectedTable: "orderFqdnSets",
		},
		{
			query:         "SELECT id, identifierType, identifierValue, registrationID, status, expires, challenges, attempted, token, validationError, validationRecord FROM authz2 WHERE\n\t\t\tregistrationID = :regID AND\n\t\t\tstatus = :status AND\n\t\t\texpires > :validUntil AND\n\t\t\tidentifierType = :dnsType AND\n\t\t\tidentifierValue = :ident\n\t\t\tORDER BY expires ASC\n\t\t\tLIMIT 1 ",
			expectedTable: "authz2",
		},
		{
			query:         "insert into `registrations` (`id`,`jwk`,`jw      k_sha256`,`contact`,`agreement`,`initialIp`,`createdAt`,`LockCol`,`status`) values (null,?,?,?,?,?,?,?,?);",
			expectedTable: "`registrations`",
		},
		{
			query:         "update `registrations` set `jwk`=?, `jwk_sh      a256`=?, `contact`=?, `agreement`=?, `initialIp`=?, `createdAt`=?, `LockCol`      =?, `status`=? where `id`=? and `LockCol`=?;",
			expectedTable: "`registrations`",
		},
		{
			query:         "SELECT COUNT(*) FROM registrations WHERE initialIP = ? AND ? < createdAt AND createdAt <= ?",
			expectedTable: "registrations",
		},
		{
			query:         "SELECT COUNT(*) FROM orders WHERE registrationID = ? AND created >= ? AND created < ?",
			expectedTable: "orders",
		},
		{
			query:         " SELECT id, identifierType, identifierValue, registrationID, status, expires, challenges, attempted, token, validationError, validationRecord FROM authz2 WHERE registrationID = ? AND status IN (?,?) AND expires > ? AND identifierType = ? AND identifierValue IN (?)",
			expectedTable: "authz2",
		},
		{
			query:         "insert into `authz2` (`id`,`identifierType`,`identifierValue`,`registrationID`,`status`,`expires`,`challenges`,`attempted`,`token`,`validationError`,`validationRecord`) values (null,?,?,?,?,?,?,?,?,?,?);",
			expectedTable: "`authz2`",
		},
		{
			query:         "insert into `orders` (`ID`,`RegistrationID`,`Expires`,`Created`,`Error`,`CertificateSerial`,`BeganProcessing`) values (null,?,?,?,?,?,?)",
			expectedTable: "`orders`",
		},
		{
			query:         "insert into `orderToAuthz2` (`OrderID`,`AuthzID`) values (?,?);",
			expectedTable: "`orderToAuthz2`",
		},
		{
			query:         "insert into `requestedNames` (`ID`,`OrderID`,`ReversedName`) values (?,?,?);",
			expectedTable: "`requestedNames`",
		},
		{
			query:         "UPDATE authz2 SET status = :status, attempted = :attempted, validationRecord = :validationRecord, validationError = :validationError, expires = :expires WHERE id = :id AND status = :pending",
			expectedTable: "authz2",
		},
		{
			query:         "insert into `precertificates` (`ID`,`Serial`,`RegistrationID`,`DER`,`Issued`,`Expires`) values (null,?,?,?,?,?);",
			expectedTable: "`precertificates`",
		},
		{
			query:         "INSERT INTO certificateStatus (serial, status, ocspLastUpdated, revokedDate, revokedReason, lastExpirationNagSent, ocspResponse, notAfter, isExpired, issuerID) VALUES (?,?,?,?,?,?,?,?,?,?)",
			expectedTable: "certificateStatus",
		},
		{
			query:         "INSERT INTO issuedNames (reversedName, serial, notBefore, renewal) VALUES (?, ?, ?, ?);",
			expectedTable: "issuedNames",
		},
		{
			query:         "insert into `certificates` (`registrationID`,`serial`,`digest`,`der`,`issued`,`expires`) values (?,?,?,?,?,?);",
			expectedTable: "`certificates`",
		},
		{
			query:         "INSERT INTO certificatesPerName (eTLDPlusOne, time, count) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE count=count+1;",
			expectedTable: "certificatesPerName",
		},
		{
			query:         "insert into `fqdnSets` (`ID`,`SetHash`,`Serial`,`Issued`,`Expires`) values (null,?,?,?,?);",
			expectedTable: "`fqdnSets`",
		},
		{
			query:         "UPDATE orders SET certificateSerial = ? WHERE id = ? AND beganProcessing = true",
			expectedTable: "orders",
		},
		{
			query:         "DELETE FROM orderFqdnSets WHERE orderID = ?",
			expectedTable: "orderFqdnSets",
		},
		{
			query:         "insert into `serials` (`ID`,`Serial`,`RegistrationID`,`Created`,`Expires`) values (null,?,?,?,?);",
			expectedTable: "`serials`",
		},
		{
			query:         "UPDATE orders SET beganProcessing = ? WHERE id = ? AND beganProcessing = ?",
			expectedTable: "orders",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("testCases.%d", i), func(t *testing.T) {
			table := tableFromQuery(tc.query)
			test.AssertEquals(t, table, tc.expectedTable)
		})
	}
}

func testDbMap(t *testing.T) *WrappedMap {
	// NOTE(@cpu): We avoid using sa.NewDBMapFromConfig here because it would
	// create a cyclic dependency. The `sa` package depends on `db` for
	// `WithTransaction`. The `db` package can't depend on the `sa` for creating
	// a DBMap. Since we only need a map for simple unit tests we can make our
	// own dbMap by hand (how artisanal).
	var config *mysql.Config
	config, err := mysql.ParseDSN(vars.DBConnSA)
	test.AssertNotError(t, err, "parsing DBConnSA DSN")

	dbConn, err := sql.Open("mysql", config.FormatDSN())
	test.AssertNotError(t, err, "opening DB connection")

	dialect := gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}
	// NOTE(@cpu): We avoid giving a sa.BoulderTypeConverter to the DbMap field to
	// avoid the cyclic dep. We don't need to convert any types in the db tests.
	dbMap := &gorp.DbMap{Db: dbConn, Dialect: dialect, TypeConverter: nil}
	return &WrappedMap{DbMap: dbMap}
}

func TestWrappedMap(t *testing.T) {
	mustDbErr := func(err error) ErrDatabaseOp {
		var dbOpErr ErrDatabaseOp
		test.AssertErrorWraps(t, err, &dbOpErr)
		return dbOpErr
	}

	testWrapper := func(dbMap Executor) {
		reg := &core.Registration{}

		// Test wrapped Get
		_, err := dbMap.Get(reg)
		test.AssertError(t, err, "expected err Getting Registration w/o type converter")
		dbOpErr := mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "get")
		test.AssertEquals(t, dbOpErr.Table, "*core.Registration")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped Insert
		err = dbMap.Insert(reg)
		test.AssertError(t, err, "expected err Inserting Registration w/o type converter")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "insert")
		test.AssertEquals(t, dbOpErr.Table, "*core.Registration")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped Update
		_, err = dbMap.Update(reg)
		test.AssertError(t, err, "expected err Updating Registration w/o type converter")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "update")
		test.AssertEquals(t, dbOpErr.Table, "*core.Registration")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped Delete
		_, err = dbMap.Delete(reg)
		test.AssertError(t, err, "expected err Deleting Registration w/o type converter")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "delete")
		test.AssertEquals(t, dbOpErr.Table, "*core.Registration")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped Select with a bogus query
		_, err = dbMap.Select(reg, "blah")
		test.AssertError(t, err, "expected err Selecting Registration w/o type converter")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "select")
		test.AssertEquals(t, dbOpErr.Table, "*core.Registration (unknown table)")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped Select with a valid query
		_, err = dbMap.Select(reg, "SELECT id, contact FROM registrationzzz WHERE id > 1;")
		test.AssertError(t, err, "expected err Selecting Registration w/o type converter")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "select")
		test.AssertEquals(t, dbOpErr.Table, "registrationzzz")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped SelectOne with a bogus query
		err = dbMap.SelectOne(reg, "blah")
		test.AssertError(t, err, "expected err SelectOne-ing Registration w/o type converter")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "select one")
		test.AssertEquals(t, dbOpErr.Table, "*core.Registration (unknown table)")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped SelectOne with a valid query
		err = dbMap.SelectOne(reg, "SELECT contact FROM doesNotExist WHERE id=1;")
		test.AssertError(t, err, "expected err SelectOne-ing Registration w/o type converter")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "select one")
		test.AssertEquals(t, dbOpErr.Table, "doesNotExist")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")

		// Test wrapped Exec
		_, err = dbMap.Exec("INSERT INTO whatever (id) VALUES (?) WHERE id = ?", 10)
		test.AssertError(t, err, "expected err Exec-ing bad query")
		dbOpErr = mustDbErr(err)
		test.AssertEquals(t, dbOpErr.Op, "exec")
		test.AssertEquals(t, dbOpErr.Table, "whatever")
		test.AssertError(t, dbOpErr.Err, "expected non-nil underlying err")
	}

	// Create a test wrapped map. It won't have a type converted registered.
	dbMap := testDbMap(t)

	// A top level WrappedMap should operate as expected with respect to wrapping
	// database errors.
	testWrapper(dbMap)

	// Using WithContext on the WrappedMap should return a map that continues to
	// operate in the expected fashion.
	dbMapWithCtx := dbMap.WithContext(context.Background())
	testWrapper(dbMapWithCtx)

	// Using Begin to start a transaction with the dbMap should return a
	// transaction that continues to operate in the expected fashion.
	tx, err := dbMap.Begin()
	defer func() { _ = tx.Rollback() }()
	test.AssertNotError(t, err, "unexpected error beginning transaction")
	testWrapper(tx)

	// Using Begin to start a transaction with the dbMap and then using
	// WithContext should return a transaction that continues to operate in the
	// expected fashion.
	tx, err = dbMap.Begin()
	defer func() { _ = tx.Rollback() }()
	test.AssertNotError(t, err, "unexpected error beginning transaction")
	txWithContext := tx.WithContext(context.Background())
	testWrapper(txWithContext)
}
