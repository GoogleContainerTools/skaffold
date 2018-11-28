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
	"time"
)

func TestTeamsService_ListComments(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/2/discussions/3/comments", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeTeamDiscussionsPreview)
		testFormValues(t, r, values{
			"direction": "desc",
		})
		fmt.Fprintf(w,
			`[
				{
					"author": {
						"login": "author",
						"id": 0,
						"avatar_url": "https://avatars1.githubusercontent.com/u/0?v=4",
						"gravatar_id": "",
						"url": "https://api.github.com/users/author",
						"html_url": "https://github.com/author",
						"followers_url": "https://api.github.com/users/author/followers",
						"following_url": "https://api.github.com/users/author/following{/other_user}",
						"gists_url": "https://api.github.com/users/author/gists{/gist_id}",
						"starred_url": "https://api.github.com/users/author/starred{/owner}{/repo}",
						"subscriptions_url": "https://api.github.com/users/author/subscriptions",
						"organizations_url": "https://api.github.com/users/author/orgs",
						"repos_url": "https://api.github.com/users/author/repos",
						"events_url": "https://api.github.com/users/author/events{/privacy}",
						"received_events_url": "https://api.github.com/users/author/received_events",
						"type": "User",
						"site_admin": false
					},
					"body": "comment",
					"body_html": "<p>comment</p>",
					"body_version": "version",
					"created_at": "2018-01-01T00:00:00Z",
					"last_edited_at": null,
					"discussion_url": "https://api.github.com/teams/2/discussions/3",
					"html_url": "https://github.com/orgs/1/teams/2/discussions/3/comments/4",
					"node_id": "node",
					"number": 4,
					"updated_at": "2018-01-01T00:00:00Z",
					"url": "https://api.github.com/teams/2/discussions/3/comments/4"
				}
			]`)
	})

	comments, _, err := client.Teams.ListComments(context.Background(), 2, 3, &DiscussionCommentListOptions{"desc"})
	if err != nil {
		t.Errorf("Teams.ListComments returned error: %v", err)
	}

	want := []*DiscussionComment{
		{
			Author: &User{
				Login:             String("author"),
				ID:                Int64(0),
				AvatarURL:         String("https://avatars1.githubusercontent.com/u/0?v=4"),
				GravatarID:        String(""),
				URL:               String("https://api.github.com/users/author"),
				HTMLURL:           String("https://github.com/author"),
				FollowersURL:      String("https://api.github.com/users/author/followers"),
				FollowingURL:      String("https://api.github.com/users/author/following{/other_user}"),
				GistsURL:          String("https://api.github.com/users/author/gists{/gist_id}"),
				StarredURL:        String("https://api.github.com/users/author/starred{/owner}{/repo}"),
				SubscriptionsURL:  String("https://api.github.com/users/author/subscriptions"),
				OrganizationsURL:  String("https://api.github.com/users/author/orgs"),
				ReposURL:          String("https://api.github.com/users/author/repos"),
				EventsURL:         String("https://api.github.com/users/author/events{/privacy}"),
				ReceivedEventsURL: String("https://api.github.com/users/author/received_events"),
				Type:              String("User"),
				SiteAdmin:         Bool(false),
			},
			Body:          String("comment"),
			BodyHTML:      String("<p>comment</p>"),
			BodyVersion:   String("version"),
			CreatedAt:     &Timestamp{time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC)},
			LastEditedAt:  nil,
			DiscussionURL: String("https://api.github.com/teams/2/discussions/3"),
			HTMLURL:       String("https://github.com/orgs/1/teams/2/discussions/3/comments/4"),
			NodeID:        String("node"),
			Number:        Int(4),
			UpdatedAt:     &Timestamp{time.Date(2018, time.January, 1, 0, 0, 0, 0, time.UTC)},
			URL:           String("https://api.github.com/teams/2/discussions/3/comments/4"),
		},
	}
	if !reflect.DeepEqual(comments, want) {
		t.Errorf("Teams.ListComments returned %+v, want %+v", comments, want)
	}
}

func TestTeamsService_GetComment(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/2/discussions/3/comments/4", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeTeamDiscussionsPreview)
		fmt.Fprint(w, `{"number":4}`)
	})

	comment, _, err := client.Teams.GetComment(context.Background(), 2, 3, 4)
	if err != nil {
		t.Errorf("Teams.GetComment returned error: %v", err)
	}

	want := &DiscussionComment{Number: Int(4)}
	if !reflect.DeepEqual(comment, want) {
		t.Errorf("Teams.GetComment returned %+v, want %+v", comment, want)
	}
}

func TestTeamsService_CreateComment(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := DiscussionComment{Body: String("c")}

	mux.HandleFunc("/teams/2/discussions/3/comments", func(w http.ResponseWriter, r *http.Request) {
		v := new(DiscussionComment)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeTeamDiscussionsPreview)
		if !reflect.DeepEqual(v, &input) {
			t.Errorf("Request body = %+v, want %+v", v, input)
		}

		fmt.Fprint(w, `{"number":4}`)
	})

	comment, _, err := client.Teams.CreateComment(context.Background(), 2, 3, input)
	if err != nil {
		t.Errorf("Teams.CreateComment returned error: %v", err)
	}

	want := &DiscussionComment{Number: Int(4)}
	if !reflect.DeepEqual(comment, want) {
		t.Errorf("Teams.CreateComment returned %+v, want %+v", comment, want)
	}
}

func TestTeamsService_EditComment(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := DiscussionComment{Body: String("e")}

	mux.HandleFunc("/teams/2/discussions/3/comments/4", func(w http.ResponseWriter, r *http.Request) {
		v := new(DiscussionComment)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "PATCH")
		testHeader(t, r, "Accept", mediaTypeTeamDiscussionsPreview)
		if !reflect.DeepEqual(v, &input) {
			t.Errorf("Request body = %+v, want %+v", v, input)
		}

		fmt.Fprint(w, `{"number":4}`)
	})

	comment, _, err := client.Teams.EditComment(context.Background(), 2, 3, 4, input)
	if err != nil {
		t.Errorf("Teams.EditComment returned error: %v", err)
	}

	want := &DiscussionComment{Number: Int(4)}
	if !reflect.DeepEqual(comment, want) {
		t.Errorf("Teams.EditComment returned %+v, want %+v", comment, want)
	}
}

func TestTeamsService_DeleteComment(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/2/discussions/3/comments/4", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testHeader(t, r, "Accept", mediaTypeTeamDiscussionsPreview)
	})

	_, err := client.Teams.DeleteComment(context.Background(), 2, 3, 4)
	if err != nil {
		t.Errorf("Teams.DeleteComment returned error: %v", err)
	}
}
