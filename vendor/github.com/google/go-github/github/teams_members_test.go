// Copyright 2018 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestTeamsService__ListTeamMembers(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/members", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		testFormValues(t, r, values{"role": "member", "page": "2"})
		fmt.Fprint(w, `[{"id":1}]`)
	})

	opt := &TeamListTeamMembersOptions{Role: "member", ListOptions: ListOptions{Page: 2}}
	members, _, err := client.Teams.ListTeamMembers(context.Background(), 1, opt)
	if err != nil {
		t.Errorf("Teams.ListTeamMembers returned error: %v", err)
	}

	want := []*User{{ID: Int64(1)}}
	if !reflect.DeepEqual(members, want) {
		t.Errorf("Teams.ListTeamMembers returned %+v, want %+v", members, want)
	}
}

func TestTeamsService__IsTeamMember_true(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/members/u", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
	})

	member, _, err := client.Teams.IsTeamMember(context.Background(), 1, "u")
	if err != nil {
		t.Errorf("Teams.IsTeamMember returned error: %v", err)
	}
	if want := true; member != want {
		t.Errorf("Teams.IsTeamMember returned %+v, want %+v", member, want)
	}
}

// ensure that a 404 response is interpreted as "false" and not an error
func TestTeamsService__IsTeamMember_false(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/members/u", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.WriteHeader(http.StatusNotFound)
	})

	member, _, err := client.Teams.IsTeamMember(context.Background(), 1, "u")
	if err != nil {
		t.Errorf("Teams.IsTeamMember returned error: %+v", err)
	}
	if want := false; member != want {
		t.Errorf("Teams.IsTeamMember returned %+v, want %+v", member, want)
	}
}

// ensure that a 400 response is interpreted as an actual error, and not simply
// as "false" like the above case of a 404
func TestTeamsService__IsTeamMember_error(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/members/u", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		http.Error(w, "BadRequest", http.StatusBadRequest)
	})

	member, _, err := client.Teams.IsTeamMember(context.Background(), 1, "u")
	if err == nil {
		t.Errorf("Expected HTTP 400 response")
	}
	if want := false; member != want {
		t.Errorf("Teams.IsTeamMember returned %+v, want %+v", member, want)
	}
}

func TestTeamsService__IsTeamMember_invalidUser(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Teams.IsTeamMember(context.Background(), 1, "%")
	testURLParseError(t, err)
}

func TestTeamsService__GetTeamMembership(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/memberships/u", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		fmt.Fprint(w, `{"url":"u", "state":"active"}`)
	})

	membership, _, err := client.Teams.GetTeamMembership(context.Background(), 1, "u")
	if err != nil {
		t.Errorf("Teams.GetTeamMembership returned error: %v", err)
	}

	want := &Membership{URL: String("u"), State: String("active")}
	if !reflect.DeepEqual(membership, want) {
		t.Errorf("Teams.GetTeamMembership returned %+v, want %+v", membership, want)
	}
}

func TestTeamsService__AddTeamMembership(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	opt := &TeamAddTeamMembershipOptions{Role: "maintainer"}

	mux.HandleFunc("/teams/1/memberships/u", func(w http.ResponseWriter, r *http.Request) {
		v := new(TeamAddTeamMembershipOptions)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "PUT")
		if !reflect.DeepEqual(v, opt) {
			t.Errorf("Request body = %+v, want %+v", v, opt)
		}

		fmt.Fprint(w, `{"url":"u", "state":"pending"}`)
	})

	membership, _, err := client.Teams.AddTeamMembership(context.Background(), 1, "u", opt)
	if err != nil {
		t.Errorf("Teams.AddTeamMembership returned error: %v", err)
	}

	want := &Membership{URL: String("u"), State: String("pending")}
	if !reflect.DeepEqual(membership, want) {
		t.Errorf("Teams.AddTeamMembership returned %+v, want %+v", membership, want)
	}
}

func TestTeamsService__RemoveTeamMembership(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/memberships/u", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Teams.RemoveTeamMembership(context.Background(), 1, "u")
	if err != nil {
		t.Errorf("Teams.RemoveTeamMembership returned error: %v", err)
	}
}
