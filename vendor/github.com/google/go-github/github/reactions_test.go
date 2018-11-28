// Copyright 2016 The go-github AUTHORS. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package github

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestReactionsService_ListCommentReactions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/comments/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"user":{"login":"l","id":2},"content":"+1"}]`))
	})

	got, _, err := client.Reactions.ListCommentReactions(context.Background(), "o", "r", 1, nil)
	if err != nil {
		t.Errorf("ListCommentReactions returned error: %v", err)
	}
	want := []*Reaction{{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListCommentReactions = %+v, want %+v", got, want)
	}
}

func TestReactionsService_CreateCommentReaction(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/comments/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"user":{"login":"l","id":2},"content":"+1"}`))
	})

	got, _, err := client.Reactions.CreateCommentReaction(context.Background(), "o", "r", 1, "+1")
	if err != nil {
		t.Errorf("CreateCommentReaction returned error: %v", err)
	}
	want := &Reaction{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateCommentReaction = %+v, want %+v", got, want)
	}
}

func TestReactionsService_ListIssueReactions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/issues/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"user":{"login":"l","id":2},"content":"+1"}]`))
	})

	got, _, err := client.Reactions.ListIssueReactions(context.Background(), "o", "r", 1, nil)
	if err != nil {
		t.Errorf("ListIssueReactions returned error: %v", err)
	}
	want := []*Reaction{{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListIssueReactions = %+v, want %+v", got, want)
	}
}

func TestReactionsService_CreateIssueReaction(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/issues/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"user":{"login":"l","id":2},"content":"+1"}`))
	})

	got, _, err := client.Reactions.CreateIssueReaction(context.Background(), "o", "r", 1, "+1")
	if err != nil {
		t.Errorf("CreateIssueReaction returned error: %v", err)
	}
	want := &Reaction{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateIssueReaction = %+v, want %+v", got, want)
	}
}

func TestReactionsService_ListIssueCommentReactions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/issues/comments/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"user":{"login":"l","id":2},"content":"+1"}]`))
	})

	got, _, err := client.Reactions.ListIssueCommentReactions(context.Background(), "o", "r", 1, nil)
	if err != nil {
		t.Errorf("ListIssueCommentReactions returned error: %v", err)
	}
	want := []*Reaction{{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListIssueCommentReactions = %+v, want %+v", got, want)
	}
}

func TestReactionsService_CreateIssueCommentReaction(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/issues/comments/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"user":{"login":"l","id":2},"content":"+1"}`))
	})

	got, _, err := client.Reactions.CreateIssueCommentReaction(context.Background(), "o", "r", 1, "+1")
	if err != nil {
		t.Errorf("CreateIssueCommentReaction returned error: %v", err)
	}
	want := &Reaction{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateIssueCommentReaction = %+v, want %+v", got, want)
	}
}

func TestReactionsService_ListPullRequestCommentReactions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pulls/comments/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"user":{"login":"l","id":2},"content":"+1"}]`))
	})

	got, _, err := client.Reactions.ListPullRequestCommentReactions(context.Background(), "o", "r", 1, nil)
	if err != nil {
		t.Errorf("ListPullRequestCommentReactions returned error: %v", err)
	}
	want := []*Reaction{{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListPullRequestCommentReactions = %+v, want %+v", got, want)
	}
}

func TestReactionsService_CreatePullRequestCommentReaction(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pulls/comments/1/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"user":{"login":"l","id":2},"content":"+1"}`))
	})

	got, _, err := client.Reactions.CreatePullRequestCommentReaction(context.Background(), "o", "r", 1, "+1")
	if err != nil {
		t.Errorf("CreatePullRequestCommentReaction returned error: %v", err)
	}
	want := &Reaction{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreatePullRequestCommentReaction = %+v, want %+v", got, want)
	}
}

func TestReactionsService_ListTeamDiscussionReactions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/discussions/2/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"user":{"login":"l","id":2},"content":"+1"}]`))
	})

	got, _, err := client.Reactions.ListTeamDiscussionReactions(context.Background(), 1, 2, nil)
	if err != nil {
		t.Errorf("ListTeamDiscussionReactions returned error: %v", err)
	}
	want := []*Reaction{{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListTeamDiscussionReactions = %+v, want %+v", got, want)
	}
}

func TestReactionsService_CreateTeamDiscussionReaction(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/discussions/2/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"user":{"login":"l","id":2},"content":"+1"}`))
	})

	got, _, err := client.Reactions.CreateTeamDiscussionReaction(context.Background(), 1, 2, "+1")
	if err != nil {
		t.Errorf("CreateTeamDiscussionReaction returned error: %v", err)
	}
	want := &Reaction{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateTeamDiscussionReaction = %+v, want %+v", got, want)
	}
}

func TestReactionService_ListTeamDiscussionCommentReactions(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/discussions/2/comments/3/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"id":1,"user":{"login":"l","id":2},"content":"+1"}]`))
	})

	got, _, err := client.Reactions.ListTeamDiscussionCommentReactions(context.Background(), 1, 2, 3, nil)
	if err != nil {
		t.Errorf("ListTeamDiscussionCommentReactions returned error: %v", err)
	}
	want := []*Reaction{{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ListTeamDiscussionCommentReactions = %+v, want %+v", got, want)
	}
}

func TestReactionService_CreateTeamDiscussionCommentReaction(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/teams/1/discussions/2/comments/3/reactions", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":1,"user":{"login":"l","id":2},"content":"+1"}`))
	})

	got, _, err := client.Reactions.CreateTeamDiscussionCommentReaction(context.Background(), 1, 2, 3, "+1")
	if err != nil {
		t.Errorf("CreateTeamDiscussionCommentReaction returned error: %v", err)
	}
	want := &Reaction{ID: Int64(1), User: &User{Login: String("l"), ID: Int64(2)}, Content: String("+1")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CreateTeamDiscussionCommentReaction = %+v, want %+v", got, want)
	}
}

func TestReactionsService_DeleteReaction(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/reactions/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
		testHeader(t, r, "Accept", mediaTypeReactionsPreview)

		w.WriteHeader(http.StatusNoContent)
	})

	if _, err := client.Reactions.DeleteReaction(context.Background(), 1); err != nil {
		t.Errorf("DeleteReaction returned error: %v", err)
	}
}
