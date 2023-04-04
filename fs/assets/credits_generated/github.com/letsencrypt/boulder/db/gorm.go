package db

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Characters allowed in an unquoted identifier by MariaDB.
// https://mariadb.com/kb/en/identifier-names/#unquoted
var mariaDBUnquotedIdentifierRE = regexp.MustCompile("^[0-9a-zA-Z$_]+$")

func validMariaDBUnquotedIdentifier(s string) error {
	if !mariaDBUnquotedIdentifierRE.MatchString(s) {
		return fmt.Errorf("invalid MariaDB identifier %q", s)
	}

	allNumeric := true
	startsNumeric := false
	for i, c := range []byte(s) {
		if c < '0' || c > '9' {
			if startsNumeric && len(s) > i && s[i] == 'e' {
				return fmt.Errorf("MariaDB identifier looks like floating point: %q", s)
			}
			allNumeric = false
			break
		}
		startsNumeric = true
	}
	if allNumeric {
		return fmt.Errorf("MariaDB identifier contains only numerals: %q", s)
	}
	return nil
}

// NewMappedSelector returns an object which can be used to automagically query
// the provided type-mapped database for rows of the parameterized type.
func NewMappedSelector[T any](executor MappedExecutor) (MappedSelector[T], error) {
	var throwaway T
	t := reflect.TypeOf(throwaway)

	// We use a very strict mapping of struct fields to table columns here:
	// - The struct must not have any embedded structs, only named fields.
	// - The struct field names must be case-insensitively identical to the
	//   column names (no struct tags necessary).
	// - The struct field names must be case-insensitively unique.
	// - Every field of the struct must correspond to a database column.
	//   - Note that the reverse is not true: it's perfectly okay for there to be
	//     database columns which do not correspond to fields in the struct; those
	//     columns will be ignored.
	// TODO: In the future, when we replace gorp's TableMap with our own, this
	// check should be performed at the time the mapping is declared.
	columns := make([]string, 0)
	seen := make(map[string]struct{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Anonymous {
			return nil, fmt.Errorf("struct contains anonymous embedded struct %q", field.Name)
		}
		column := strings.ToLower(t.Field(i).Name)
		err := validMariaDBUnquotedIdentifier(column)
		if err != nil {
			return nil, fmt.Errorf("struct field maps to unsafe db column name %q", column)
		}
		if _, found := seen[column]; found {
			return nil, fmt.Errorf("struct fields map to duplicate column name %q", column)
		}
		seen[column] = struct{}{}
		columns = append(columns, column)
	}

	return &mappedSelector[T]{wrapped: executor, columns: columns}, nil
}

type mappedSelector[T any] struct {
	wrapped MappedExecutor
	columns []string
}

// Query performs a SELECT on the appropriate table for T. It combines the best
// features of gorp, the go stdlib, and generics, using the type parameter of
// the typeSelector object to automatically look up the proper table name and
// columns to select. It returns an iterable which yields fully-populated
// objects of the parameterized type directly. The given clauses MUST be only
// the bits of a sql query from "WHERE ..." onwards; if they contain any of the
// "SELECT ... FROM ..." portion of the query it will result in an error. The
// args take the same kinds of values as gorp's SELECT: either one argument per
// positional placeholder, or a map of placeholder names to their arguments
// (see https://pkg.go.dev/gopkg.in/gorp.v2#readme-ad-hoc-sql).
//
// The caller is responsible for calling `Rows.Close()` when they are done with
// the query. The caller is also responsible for ensuring that the clauses
// argument does not contain any user-influenced input.
func (ts mappedSelector[T]) Query(ctx context.Context, clauses string, args ...interface{}) (Rows[T], error) {
	// Look up the table to use based on the type of this TypeSelector.
	var throwaway T
	tableMap, err := ts.wrapped.TableFor(reflect.TypeOf(throwaway), false)
	if err != nil {
		return nil, fmt.Errorf("database model type not mapped to table name: %w", err)
	}

	return ts.QueryFrom(ctx, tableMap.TableName, clauses, args...)
}

// QueryFrom is the same as Query, but it additionally takes a table name to
// select from, rather than automatically computing the table name from gorp's
// DbMap.
//
// The caller is responsible for calling `Rows.Close()` when they are done with
// the query. The caller is also responsible for ensuring that the clauses
// argument does not contain any user-influenced input.
func (ts mappedSelector[T]) QueryFrom(ctx context.Context, tablename string, clauses string, args ...interface{}) (Rows[T], error) {
	err := validMariaDBUnquotedIdentifier(tablename)
	if err != nil {
		return nil, err
	}

	// Construct the query from the column names, table name, and given clauses.
	// Note that the column names here are in the order given by
	query := fmt.Sprintf(
		"SELECT %s FROM %s %s",
		strings.Join(ts.columns, ", "),
		tablename,
		clauses,
	)

	r, err := ts.wrapped.WithContext(ctx).Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("reading db: %w", err)
	}

	return &rows[T]{wrapped: r, numCols: len(ts.columns)}, nil
}

// rows is a wrapper around the stdlib's sql.rows, but with a more
// type-safe method to get actual row content.
type rows[T any] struct {
	wrapped *sql.Rows
	numCols int
}

// Next is a wrapper around sql.Rows.Next(). It must be called before every call
// to Get(), including the first.
func (r rows[T]) Next() bool {
	return r.wrapped.Next()
}

// Get is a wrapper around sql.Rows.Scan(). Rather than populating an arbitrary
// number of &interface{} arguments, it returns a populated object of the
// parameterized type.
func (r rows[T]) Get() (*T, error) {
	result := new(T)
	v := reflect.ValueOf(result)

	// Because sql.Rows.Scan(...) takes a variadic number of individual targets to
	// read values into, build a slice that can be splatted into the call. Use the
	// pre-computed list of in-order column names to populate it.
	scanTargets := make([]interface{}, r.numCols)
	for i := range scanTargets {
		field := v.Elem().Field(i)
		scanTargets[i] = field.Addr().Interface()
	}

	err := r.wrapped.Scan(scanTargets...)
	if err != nil {
		return nil, fmt.Errorf("reading db row: %w", err)
	}

	return result, nil
}

// Err is a wrapper around sql.Rows.Err(). It should be checked immediately
// after Next() returns false for any reason.
func (r rows[T]) Err() error {
	return r.wrapped.Err()
}

// Close is a wrapper around sql.Rows.Close(). It must be called when the caller
// is done reading rows, regardless of success or error.
func (r rows[T]) Close() error {
	return r.wrapped.Close()
}
