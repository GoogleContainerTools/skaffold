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
	"strings"
	"testing"
	"time"
)

func TestTeamsService_ListTeams(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/orgs/o/teams", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		testFormValues(t, r, values{"page": "2"})
		fmt.Fprint(w, `[{"id":1}]`)
	})

	opt := &ListOptions{Page: 2}
	teams, _, err := client.Teams.ListTeams(context.Background(), "o", opt)
	if err != nil {
		t.Errorf("Teams.ListTeams returned error: %v", err)
	}

	want := []*Team{{ID: Int64(1)}}
	if !reflect.DeepEqual(teams, want) {
		t.Errorf("Teams.ListTeams returned %+v, want %+v", teams, want)
	}
}

func TestTeamsService_ListTeams_invalidOrg(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Teams.ListTeams(context.Background(), "%", nil)
	testURLParseError(t, err)
}

func TestTeamsService_GetTeam(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		fmt.Fprint(w, `{"id":1, "name":"n", "description": "d", "url":"u", "slug": "s", "permission":"p", "ldap_dn":"cn=n,ou=groups,dc=example,dc=com", "parent":null}`)
	})

	team, _, err := client.Teams.GetTeam(context.Background(), 1)
	if err != nil {
		t.Errorf("Teams.GetTeam returned error: %v", err)
	}

	want := &Team{ID: Int64(1), Name: String("n"), Description: String("d"), URL: String("u"), Slug: String("s"), Permission: String("p"), LDAPDN: String("cn=n,ou=groups,dc=example,dc=com")}
	if !reflect.DeepEqual(team, want) {
		t.Errorf("Teams.GetTeam returned %+v, want %+v", team, want)
	}
}

func TestTeamsService_GetTeam_nestedTeams(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		fmt.Fprint(w, `{"id":1, "name":"n", "description": "d", "url":"u", "slug": "s", "permission":"p",
		"parent": {"id":2, "name":"n", "description": "d", "parent": null}}`)
	})

	team, _, err := client.Teams.GetTeam(context.Background(), 1)
	if err != nil {
		t.Errorf("Teams.GetTeam returned error: %v", err)
	}

	want := &Team{ID: Int64(1), Name: String("n"), Description: String("d"), URL: String("u"), Slug: String("s"), Permission: String("p"),
		Parent: &Team{ID: Int64(2), Name: String("n"), Description: String("d")},
	}
	if !reflect.DeepEqual(team, want) {
		t.Errorf("Teams.GetTeam returned %+v, want %+v", team, want)
	}
}

func TestTeamsService_CreateTeam(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := NewTeam{Name: "n", Privacy: String("closed"), RepoNames: []string{"r"}}

	mux.HandleFunc("/orgs/o/teams", func(w http.ResponseWriter, r *http.Request) {
		v := new(NewTeam)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		if !reflect.DeepEqual(v, &input) {
			t.Errorf("Request body = %+v, want %+v", v, input)
		}

		fmt.Fprint(w, `{"id":1}`)
	})

	team, _, err := client.Teams.CreateTeam(context.Background(), "o", input)
	if err != nil {
		t.Errorf("Teams.CreateTeam returned error: %v", err)
	}

	want := &Team{ID: Int64(1)}
	if !reflect.DeepEqual(team, want) {
		t.Errorf("Teams.CreateTeam returned %+v, want %+v", team, want)
	}
}

func TestTeamsService_CreateTeam_invalidOrg(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Teams.CreateTeam(context.Background(), "%", NewTeam{})
	testURLParseError(t, err)
}

func TestTeamsService_EditTeam(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := NewTeam{Name: "n", Privacy: String("closed")}

	mux.HandleFunc("/teams/1", func(w http.ResponseWriter, r *http.Request) {
		v := new(NewTeam)
		json.NewDecoder(r.Body).Decode(v)

		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		testMethod(t, r, "PATCH")
		if !reflect.DeepEqual(v, &input) {
			t.Errorf("Request body = %+v, want %+v", v, input)
		}

		fmt.Fprint(w, `{"id":1}`)
	})

	team, _, err := client.Teams.EditTeam(context.Background(), 1, input)
	if err != nil {
		t.Errorf("Teams.EditTeam returned error: %v", err)
	}

	want := &Team{ID: Int64(1)}
	if !reflect.DeepEqual(team, want) {
		t.Errorf("Teams.EditTeam returned %+v, want %+v", team, want)
	}
}

func TestTeamsService_DeleteTeam(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
	})

	_, err := client.Teams.DeleteTeam(context.Background(), 1)
	if err != nil {
		t.Errorf("Teams.DeleteTeam returned error: %v", err)
	}
}

func TestTeamsService_ListChildTeams(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/teams", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		testFormValues(t, r, values{"page": "2"})
		fmt.Fprint(w, `[{"id":2}]`)
	})

	opt := &ListOptions{Page: 2}
	teams, _, err := client.Teams.ListChildTeams(context.Background(), 1, opt)
	if err != nil {
		t.Errorf("Teams.ListTeams returned error: %v", err)
	}

	want := []*Team{{ID: Int64(2)}}
	if !reflect.DeepEqual(teams, want) {
		t.Errorf("Teams.ListTeams returned %+v, want %+v", teams, want)
	}
}

func TestTeamsService_ListTeamRepos(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/repos", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		acceptHeaders := []string{mediaTypeTopicsPreview, mediaTypeNestedTeamsPreview}
		testHeader(t, r, "Accept", strings.Join(acceptHeaders, ", "))
		testFormValues(t, r, values{"page": "2"})
		fmt.Fprint(w, `[{"id":1}]`)
	})

	opt := &ListOptions{Page: 2}
	members, _, err := client.Teams.ListTeamRepos(context.Background(), 1, opt)
	if err != nil {
		t.Errorf("Teams.ListTeamRepos returned error: %v", err)
	}

	want := []*Repository{{ID: Int64(1)}}
	if !reflect.DeepEqual(members, want) {
		t.Errorf("Teams.ListTeamRepos returned %+v, want %+v", members, want)
	}
}

func TestTeamsService_IsTeamRepo_true(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		acceptHeaders := []string{mediaTypeOrgPermissionRepo, mediaTypeNestedTeamsPreview}
		testHeader(t, r, "Accept", strings.Join(acceptHeaders, ", "))
		fmt.Fprint(w, `{"id":1}`)
	})

	repo, _, err := client.Teams.IsTeamRepo(context.Background(), 1, "o", "r")
	if err != nil {
		t.Errorf("Teams.IsTeamRepo returned error: %v", err)
	}

	want := &Repository{ID: Int64(1)}
	if !reflect.DeepEqual(repo, want) {
		t.Errorf("Teams.IsTeamRepo returned %+v, want %+v", repo, want)
	}
}

func TestTeamsService_IsTeamRepo_false(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		w.WriteHeader(http.StatusNotFound)
	})

	repo, resp, err := client.Teams.IsTeamRepo(context.Background(), 1, "o", "r")
	if err == nil {
		t.Errorf("Expected HTTP 404 response")
	}
	if got, want := resp.Response.StatusCode, http.StatusNotFound; got != want {
		t.Errorf("Teams.IsTeamRepo returned status %d, want %d", got, want)
	}
	if repo != nil {
		t.Errorf("Teams.IsTeamRepo returned %+v, want nil", repo)
	}
}

func TestTeamsService_IsTeamRepo_error(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		http.Error(w, "BadRequest", http.StatusBadRequest)
	})

	repo, resp, err := client.Teams.IsTeamRepo(context.Background(), 1, "o", "r")
	if err == nil {
		t.Errorf("Expected HTTP 400 response")
	}
	if got, want := resp.Response.StatusCode, http.StatusBadRequest; got != want {
		t.Errorf("Teams.IsTeamRepo returned status %d, want %d", got, want)
	}
	if repo != nil {
		t.Errorf("Teams.IsTeamRepo returned %+v, want nil", repo)
	}
}

func TestTeamsService_IsTeamRepo_invalidOwner(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Teams.IsTeamRepo(context.Background(), 1, "%", "r")
	testURLParseError(t, err)
}

func TestTeamsService_AddTeamRepo(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	opt := &TeamAddTeamRepoOptions{Permission: "admin"}

	mux.HandleFunc("/teams/1/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		v := new(TeamAddTeamRepoOptions)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "PUT")
		if !reflect.DeepEqual(v, opt) {
			t.Errorf("Request body = %+v, want %+v", v, opt)
		}

		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Teams.AddTeamRepo(context.Background(), 1, "o", "r", opt)
	if err != nil {
		t.Errorf("Teams.AddTeamRepo returned error: %v", err)
	}
}

func TestTeamsService_AddTeamRepo_noAccess(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PUT")
		w.WriteHeader(http.StatusUnprocessableEntity)
	})

	_, err := client.Teams.AddTeamRepo(context.Background(), 1, "o", "r", nil)
	if err == nil {
		t.Errorf("Expcted error to be returned")
	}
}

func TestTeamsService_AddTeamRepo_invalidOwner(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, err := client.Teams.AddTeamRepo(context.Background(), 1, "%", "r", nil)
	testURLParseError(t, err)
}

func TestTeamsService_RemoveTeamRepo(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/repos/o/r", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		w.WriteHeader(http.StatusNoContent)
	})

	_, err := client.Teams.RemoveTeamRepo(context.Background(), 1, "o", "r")
	if err != nil {
		t.Errorf("Teams.RemoveTeamRepo returned error: %v", err)
	}
}

func TestTeamsService_RemoveTeamRepo_invalidOwner(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, err := client.Teams.RemoveTeamRepo(context.Background(), 1, "%", "r")
	testURLParseError(t, err)
}

func TestTeamsService_ListUserTeams(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/user/teams", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeNestedTeamsPreview)
		testFormValues(t, r, values{"page": "1"})
		fmt.Fprint(w, `[{"id":1}]`)
	})

	opt := &ListOptions{Page: 1}
	teams, _, err := client.Teams.ListUserTeams(context.Background(), opt)
	if err != nil {
		t.Errorf("Teams.ListUserTeams returned error: %v", err)
	}

	want := []*Team{{ID: Int64(1)}}
	if !reflect.DeepEqual(teams, want) {
		t.Errorf("Teams.ListUserTeams returned %+v, want %+v", teams, want)
	}
}

func TestTeamsService_ListPendingTeamInvitations(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/invitations", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testFormValues(t, r, values{"page": "1"})
		fmt.Fprint(w, `[
				{
    					"id": 1,
    					"login": "monalisa",
    					"email": "octocat@github.com",
    					"role": "direct_member",
    					"created_at": "2017-01-21T00:00:00Z",
    					"inviter": {
      						"login": "other_user",
      						"id": 1,
      						"avatar_url": "https://github.com/images/error/other_user_happy.gif",
      						"gravatar_id": "",
      						"url": "https://api.github.com/users/other_user",
      						"html_url": "https://github.com/other_user",
      						"followers_url": "https://api.github.com/users/other_user/followers",
      						"following_url": "https://api.github.com/users/other_user/following/other_user",
      						"gists_url": "https://api.github.com/users/other_user/gists/gist_id",
      						"starred_url": "https://api.github.com/users/other_user/starred/owner/repo",
      						"subscriptions_url": "https://api.github.com/users/other_user/subscriptions",
      						"organizations_url": "https://api.github.com/users/other_user/orgs",
      						"repos_url": "https://api.github.com/users/other_user/repos",
      						"events_url": "https://api.github.com/users/other_user/events/privacy",
      						"received_events_url": "https://api.github.com/users/other_user/received_events/privacy",
      						"type": "User",
      						"site_admin": false
    					}
  				}
			]`)
	})

	opt := &ListOptions{Page: 1}
	invitations, _, err := client.Teams.ListPendingTeamInvitations(context.Background(), 1, opt)
	if err != nil {
		t.Errorf("Teams.ListPendingTeamInvitations returned error: %v", err)
	}

	createdAt := time.Date(2017, time.January, 21, 0, 0, 0, 0, time.UTC)
	want := []*Invitation{
		{
			ID:        Int64(1),
			Login:     String("monalisa"),
			Email:     String("octocat@github.com"),
			Role:      String("direct_member"),
			CreatedAt: &createdAt,
			Inviter: &User{
				Login:             String("other_user"),
				ID:                Int64(1),
				AvatarURL:         String("https://github.com/images/error/other_user_happy.gif"),
				GravatarID:        String(""),
				URL:               String("https://api.github.com/users/other_user"),
				HTMLURL:           String("https://github.com/other_user"),
				FollowersURL:      String("https://api.github.com/users/other_user/followers"),
				FollowingURL:      String("https://api.github.com/users/other_user/following/other_user"),
				GistsURL:          String("https://api.github.com/users/other_user/gists/gist_id"),
				StarredURL:        String("https://api.github.com/users/other_user/starred/owner/repo"),
				SubscriptionsURL:  String("https://api.github.com/users/other_user/subscriptions"),
				OrganizationsURL:  String("https://api.github.com/users/other_user/orgs"),
				ReposURL:          String("https://api.github.com/users/other_user/repos"),
				EventsURL:         String("https://api.github.com/users/other_user/events/privacy"),
				ReceivedEventsURL: String("https://api.github.com/users/other_user/received_events/privacy"),
				Type:              String("User"),
				SiteAdmin:         Bool(false),
			},
		}}

	if !reflect.DeepEqual(invitations, want) {
		t.Errorf("Teams.ListPendingTeamInvitations returned %+v, want %+v", invitations, want)
	}
}
