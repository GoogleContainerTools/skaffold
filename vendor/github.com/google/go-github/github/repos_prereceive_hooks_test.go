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

func TestRepositoriesService_ListPreReceiveHooks(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pre-receive-hooks", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypePreReceiveHooksPreview)
		testFormValues(t, r, values{"page": "2"})
		fmt.Fprint(w, `[{"id":1}, {"id":2}]`)
	})

	opt := &ListOptions{Page: 2}

	hooks, _, err := client.Repositories.ListPreReceiveHooks(context.Background(), "o", "r", opt)
	if err != nil {
		t.Errorf("Repositories.ListHooks returned error: %v", err)
	}

	want := []*PreReceiveHook{{ID: Int64(1)}, {ID: Int64(2)}}
	if !reflect.DeepEqual(hooks, want) {
		t.Errorf("Repositories.ListPreReceiveHooks returned %+v, want %+v", hooks, want)
	}
}

func TestRepositoriesService_ListPreReceiveHooks_invalidOwner(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Repositories.ListPreReceiveHooks(context.Background(), "%", "%", nil)
	testURLParseError(t, err)
}

func TestRepositoriesService_GetPreReceiveHook(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pre-receive-hooks/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypePreReceiveHooksPreview)
		fmt.Fprint(w, `{"id":1}`)
	})

	hook, _, err := client.Repositories.GetPreReceiveHook(context.Background(), "o", "r", 1)
	if err != nil {
		t.Errorf("Repositories.GetPreReceiveHook returned error: %v", err)
	}

	want := &PreReceiveHook{ID: Int64(1)}
	if !reflect.DeepEqual(hook, want) {
		t.Errorf("Repositories.GetPreReceiveHook returned %+v, want %+v", hook, want)
	}
}

func TestRepositoriesService_GetPreReceiveHook_invalidOwner(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Repositories.GetPreReceiveHook(context.Background(), "%", "%", 1)
	testURLParseError(t, err)
}

func TestRepositoriesService_UpdatePreReceiveHook(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	input := &PreReceiveHook{Name: String("t")}

	mux.HandleFunc("/repos/o/r/pre-receive-hooks/1", func(w http.ResponseWriter, r *http.Request) {
		v := new(PreReceiveHook)
		json.NewDecoder(r.Body).Decode(v)

		testMethod(t, r, "PATCH")
		if !reflect.DeepEqual(v, input) {
			t.Errorf("Request body = %+v, want %+v", v, input)
		}

		fmt.Fprint(w, `{"id":1}`)
	})

	hook, _, err := client.Repositories.UpdatePreReceiveHook(context.Background(), "o", "r", 1, input)
	if err != nil {
		t.Errorf("Repositories.UpdatePreReceiveHook returned error: %v", err)
	}

	want := &PreReceiveHook{ID: Int64(1)}
	if !reflect.DeepEqual(hook, want) {
		t.Errorf("Repositories.UpdatePreReceiveHook returned %+v, want %+v", hook, want)
	}
}

func TestRepositoriesService_PreReceiveHook_invalidOwner(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, _, err := client.Repositories.UpdatePreReceiveHook(context.Background(), "%", "%", 1, nil)
	testURLParseError(t, err)
}

func TestRepositoriesService_DeletePreReceiveHook(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/pre-receive-hooks/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "DELETE")
	})

	_, err := client.Repositories.DeletePreReceiveHook(context.Background(), "o", "r", 1)
	if err != nil {
		t.Errorf("Repositories.DeletePreReceiveHook returned error: %v", err)
	}
}

func TestRepositoriesService_DeletePreReceiveHook_invalidOwner(t *testing.T) {
	client, _, _, teardown := setup()
	defer teardown()

	_, err := client.Repositories.DeletePreReceiveHook(context.Background(), "%", "%", 1)
	testURLParseError(t, err)
}
