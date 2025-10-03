// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package database

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"cloud.google.com/go/longrunning/autogen/longrunningpb"
	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/googleapis/gax-go/v2"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var retryer = gax.OnCodes(
	[]codes.Code{codes.DeadlineExceeded, codes.Unavailable},
	gax.Backoff{Initial: time.Millisecond, Max: time.Millisecond, Multiplier: 1.0},
)

// CreateDatabaseWithRetry creates a new database and retries the call if the
// backend returns a retryable error. The actual CreateDatabase RPC is only
// retried if the initial call did not reach the server. In other cases, the
// client will query the backend for the long-running operation that was
// created by the initial RPC and return that operation.
func (c *DatabaseAdminClient) CreateDatabaseWithRetry(ctx context.Context, req *databasepb.CreateDatabaseRequest, opts ...gax.CallOption) (*CreateDatabaseOperation, error) {
	for {
		db, createErr := c.CreateDatabase(ctx, req, opts...)
		if createErr == nil {
			return db, nil
		}
		// Failed, check whether we should retry.
		delay, shouldRetry := retryer.Retry(createErr)
		if !shouldRetry {
			return nil, createErr
		}
		if err := gax.Sleep(ctx, delay); err != nil {
			return nil, err
		}
		// Extract the name of the database.
		dbName := extractDBName(req.CreateStatement)
		// Query the backend for any corresponding long-running operation to
		// determine whether we should retry the RPC or not.
		iter := c.ListDatabaseOperations(ctx, &databasepb.ListDatabaseOperationsRequest{
			Parent: req.Parent,
			Filter: fmt.Sprintf("(metadata.@type:type.googleapis.com/google.spanner.admin.database.v1.CreateDatabaseMetadata) AND (name:%s/databases/%s/operations/)", req.Parent, dbName),
		}, opts...)
		var mostRecentOp *longrunningpb.Operation
		for {
			op, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return nil, err
			}
			// A running operation is the most recent and should be returned.
			if !op.Done {
				return c.CreateDatabaseOperation(op.Name), nil
			}
			if op.GetError() == nil {
				mostRecentOp = op
			}
		}
		if mostRecentOp == nil {
			continue
		}
		// Only finished operations found. Check whether the database exists.
		_, getErr := c.GetDatabase(ctx, &databasepb.GetDatabaseRequest{
			Name: fmt.Sprintf("%s/databases/%s", req.Parent, dbName),
		})
		if getErr == nil {
			// Database found, return one of the long-running operations that
			// has finished, which again should return the database.
			return c.CreateDatabaseOperation(mostRecentOp.Name), nil
		}
		if status.Code(getErr) == codes.NotFound {
			continue
		}
		// Error getting the database that was not NotFound.
		return nil, getErr
	}
}

var dbNameRegEx = regexp.MustCompile("\\s*CREATE\\s+DATABASE\\s+(.+)\\s*")

// extractDBName extracts the database name from a valid CREATE DATABASE <db>
// statement. We don't have to worry about invalid create statements, as those
// should already have been handled by the backend and should return a non-
// retryable error.
func extractDBName(createStatement string) string {
	if dbNameRegEx.MatchString(createStatement) {
		namePossiblyWithQuotes := strings.TrimRightFunc(dbNameRegEx.FindStringSubmatch(createStatement)[1], unicode.IsSpace)
		if len(namePossiblyWithQuotes) > 0 && namePossiblyWithQuotes[0] == '`' {
			if len(namePossiblyWithQuotes) > 5 && namePossiblyWithQuotes[1] == '`' && namePossiblyWithQuotes[2] == '`' {
				return string(namePossiblyWithQuotes[3 : len(namePossiblyWithQuotes)-3])
			}
			return string(namePossiblyWithQuotes[1 : len(namePossiblyWithQuotes)-1])
		}
		return string(namePossiblyWithQuotes)
	}
	return ""
}
