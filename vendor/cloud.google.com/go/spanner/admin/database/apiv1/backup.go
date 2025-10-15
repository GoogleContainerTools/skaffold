/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package database

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"cloud.google.com/go/spanner/admin/database/apiv1/databasepb"
	"github.com/googleapis/gax-go/v2"
	pbt "google.golang.org/protobuf/types/known/timestamppb"
)

var (
	validDBPattern = regexp.MustCompile("^projects/(?P<project>[^/]+)/instances/(?P<instance>[^/]+)/databases/(?P<database>[^/]+)$")
)

// StartBackupOperation creates a backup of the given database. It will be stored
// as projects/<project>/instances/<instance>/backups/<backupID>. The
// backup will be automatically deleted by Cloud Spanner after its expiration.
//
// backupID must be unique across an instance.
//
// expireTime is the time the backup will expire. It is respected to
// microsecond granularity.
//
// databasePath must have the form
// projects/<project>/instances/<instance>/databases/<database>.
func (c *DatabaseAdminClient) StartBackupOperation(ctx context.Context, backupID string, databasePath string, expireTime time.Time, opts ...gax.CallOption) (*CreateBackupOperation, error) {
	m := validDBPattern.FindStringSubmatch(databasePath)
	if m == nil {
		return nil, fmt.Errorf("database name %q should conform to pattern %q",
			databasePath, validDBPattern)
	}
	ts := &pbt.Timestamp{Seconds: expireTime.Unix(), Nanos: int32(expireTime.Nanosecond())}
	// Create request from parameters.
	req := &databasepb.CreateBackupRequest{
		Parent:   fmt.Sprintf("projects/%s/instances/%s", m[1], m[2]),
		BackupId: backupID,
		Backup: &databasepb.Backup{
			Database:   databasePath,
			ExpireTime: ts,
		},
	}
	return c.CreateBackup(ctx, req, opts...)
}
