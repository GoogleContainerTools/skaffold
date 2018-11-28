// Copyright 2018 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestMigrationService_StartUserMigration(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/migrations", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeMigrationsPreview)

		w.WriteHeader(http.StatusCreated)
		w.Write(userMigrationJSON)
	})

	opt := &UserMigrationOptions{
		LockRepositories:   true,
		ExcludeAttachments: false,
	}

	got, _, err := client.Migrations.StartUserMigration(context.Background(), []string{"r"}, opt)
	if err != nil {
		t.Errorf("StartUserMigration returned error: %v", err)
	}

	want := wantUserMigration
	if !reflect.DeepEqual(want, got) {
		t.Errorf("StartUserMigration = %v, want = %v", got, want)
	}
}

func TestMigrationService_ListUserMigrations(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/migrations", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeMigrationsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("[%s]", userMigrationJSON)))
	})

	got, _, err := client.Migrations.ListUserMigrations(context.Background())
	if err != nil {
		t.Errorf("ListUserMigrations returned error %v", err)
	}

	want := []*UserMigration{wantUserMigration}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("ListUserMigrations = %v, want = %v", got, want)
	}
}

func TestMigrationService_UserMigrationStatus(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/migrations/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeMigrationsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write(userMigrationJSON)
	})

	got, _, err := client.Migrations.UserMigrationStatus(context.Background(), 1)
	if err != nil {
		t.Errorf("UserMigrationStatus returned error %v", err)
	}

	want := wantUserMigration
	if !reflect.DeepEqual(want, got) {
		t.Errorf("UserMigrationStatus = %v, want = %v", got, want)
	}
}

func TestMigrationService_UserMigrationArchiveURL(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/migrations/1/archive", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeMigrationsPreview)

		http.Redirect(w, r, "/go-github", http.StatusFound)
	})

	mux.HandleFunc("/go-github", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")

		w.WriteHeader(http.StatusOK)
	})

	got, err := client.Migrations.UserMigrationArchiveURL(context.Background(), 1)
	if err != nil {
		t.Errorf("UserMigrationArchiveURL returned error %v", err)
	}

	want := "/go-github"
	if !strings.HasSuffix(got, want) {
		t.Errorf("UserMigrationArchiveURL = %v, want = %v", got, want)
	}
}

func TestMigrationService_DeleteUserMigration(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/migrations/1/archive", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testHeader(t, r, "Accept", mediaTypeMigrationsPreview)

		w.WriteHeader(http.StatusNoContent)
	})

	got, err := client.Migrations.DeleteUserMigration(context.Background(), 1)
	if err != nil {
		t.Errorf("DeleteUserMigration returned error %v", err)
	}

	if got.StatusCode != http.StatusNoContent {
		t.Errorf("DeleteUserMigration returned status = %v, want = %v", got.StatusCode, http.StatusNoContent)
	}
}

func TestMigrationService_UnlockUserRepo(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/migrations/1/repos/r/lock", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testHeader(t, r, "Accept", mediaTypeMigrationsPreview)

		w.WriteHeader(http.StatusNoContent)
	})

	got, err := client.Migrations.UnlockUserRepo(context.Background(), 1, "r")
	if err != nil {
		t.Errorf("UnlockUserRepo returned error %v", err)
	}

	if got.StatusCode != http.StatusNoContent {
		t.Errorf("UnlockUserRepo returned status = %v, want = %v", got.StatusCode, http.StatusNoContent)
	}
}

var userMigrationJSON = []byte(`{
  "id": 79,
  "guid": "0b989ba4-242f-11e5-81e1-c7b6966d2516",
  "state": "pending",
  "lock_repositories": true,
  "exclude_attachments": false,
  "url": "https://api.github.com/orgs/octo-org/migrations/79",
  "created_at": "2015-07-06T15:33:38-07:00",
  "updated_at": "2015-07-06T15:33:38-07:00",
  "repositories": [
    {
      "id": 1296269,
      "name": "Hello-World",
      "full_name": "octocat/Hello-World",
      "description": "This your first repo!"
    }
  ]
}`)

var wantUserMigration = &UserMigration{
	ID:                 Int64(79),
	GUID:               String("0b989ba4-242f-11e5-81e1-c7b6966d2516"),
	State:              String("pending"),
	LockRepositories:   Bool(true),
	ExcludeAttachments: Bool(false),
	URL:                String("https://api.github.com/orgs/octo-org/migrations/79"),
	CreatedAt:          String("2015-07-06T15:33:38-07:00"),
	UpdatedAt:          String("2015-07-06T15:33:38-07:00"),
	Repositories: []*Repository{
		{
			ID:          Int64(1296269),
			Name:        String("Hello-World"),
			FullName:    String("octocat/Hello-World"),
			Description: String("This your first repo!"),
		},
	},
}
