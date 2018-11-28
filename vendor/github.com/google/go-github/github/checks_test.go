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
	"testing"
	"time"
)

func TestChecksService_GetCheckRun(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-runs/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		fmt.Fprint(w, `{
			"id": 1,
                        "name":"testCheckRun",
			"status": "completed",
			"conclusion": "neutral",
			"started_at": "2018-05-04T01:14:52Z",
			"completed_at": "2018-05-04T01:14:52Z"}`)
	})
	checkRun, _, err := client.Checks.GetCheckRun(context.Background(), "o", "r", 1)
	if err != nil {
		t.Errorf("Checks.GetCheckRun return error: %v", err)
	}
	startedAt, _ := time.Parse(time.RFC3339, "2018-05-04T01:14:52Z")
	completeAt, _ := time.Parse(time.RFC3339, "2018-05-04T01:14:52Z")

	want := &CheckRun{
		ID:          Int64(1),
		Status:      String("completed"),
		Conclusion:  String("neutral"),
		StartedAt:   &Timestamp{startedAt},
		CompletedAt: &Timestamp{completeAt},
		Name:        String("testCheckRun"),
	}
	if !reflect.DeepEqual(checkRun, want) {
		t.Errorf("Checks.GetCheckRun return %+v, want %+v", checkRun, want)
	}
}

func TestChecksService_GetCheckSuite(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-suites/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		fmt.Fprint(w, `{
			"id": 1,
                        "head_branch":"master",
			"head_sha": "deadbeef",
			"conclusion": "neutral",
                        "before": "deadbeefb",
                        "after": "deadbeefa",
			"status": "completed"}`)
	})
	checkSuite, _, err := client.Checks.GetCheckSuite(context.Background(), "o", "r", 1)
	if err != nil {
		t.Errorf("Checks.GetCheckSuite return error: %v", err)
	}
	want := &CheckSuite{
		ID:         Int64(1),
		HeadBranch: String("master"),
		HeadSHA:    String("deadbeef"),
		AfterSHA:   String("deadbeefa"),
		BeforeSHA:  String("deadbeefb"),
		Status:     String("completed"),
		Conclusion: String("neutral"),
	}
	if !reflect.DeepEqual(checkSuite, want) {
		t.Errorf("Checks.GetCheckSuite return %+v, want %+v", checkSuite, want)
	}
}

func TestChecksService_CreateCheckRun(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-runs", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		fmt.Fprint(w, `{
			"id": 1,
                        "name":"testCreateCheckRun",
                        "head_sha":"deadbeef",
			"status": "in_progress",
			"conclusion": null,
			"started_at": "2018-05-04T01:14:52Z",
			"completed_at": null,
                        "output":{"title": "Mighty test report", "summary":"", "text":""}}`)
	})
	startedAt, _ := time.Parse(time.RFC3339, "2018-05-04T01:14:52Z")
	checkRunOpt := CreateCheckRunOptions{
		HeadBranch: "master",
		Name:       "testCreateCheckRun",
		HeadSHA:    "deadbeef",
		Status:     String("in_progress"),
		StartedAt:  &Timestamp{startedAt},
		Output: &CheckRunOutput{
			Title:   String("Mighty test report"),
			Summary: String(""),
			Text:    String(""),
		},
	}

	checkRun, _, err := client.Checks.CreateCheckRun(context.Background(), "o", "r", checkRunOpt)
	if err != nil {
		t.Errorf("Checks.CreateCheckRun return error: %v", err)
	}

	want := &CheckRun{
		ID:        Int64(1),
		Status:    String("in_progress"),
		StartedAt: &Timestamp{startedAt},
		HeadSHA:   String("deadbeef"),
		Name:      String("testCreateCheckRun"),
		Output: &CheckRunOutput{
			Title:   String("Mighty test report"),
			Summary: String(""),
			Text:    String(""),
		},
	}
	if !reflect.DeepEqual(checkRun, want) {
		t.Errorf("Checks.CreateCheckRun return %+v, want %+v", checkRun, want)
	}
}

func TestChecksService_ListCheckRunAnnotations(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-runs/1/annotations", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		testFormValues(t, r, values{
			"page": "1",
		})
		fmt.Fprint(w, `[{
		                           "path": "README.md",
		                           "blob_href": "https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/README.md",
		                           "start_line": 2,
		                           "end_line": 2,
		                           "annotation_level": "warning",
		                           "message": "Check your spelling for 'banaas'.",
                                           "title": "Spell check",
		                           "raw_details": "Do you mean 'bananas' or 'banana'?"}]`,
		)
	})

	checkRunAnnotations, _, err := client.Checks.ListCheckRunAnnotations(context.Background(), "o", "r", 1, &ListOptions{Page: 1})
	if err != nil {
		t.Errorf("Checks.ListCheckRunAnnotations return error: %v", err)
	}

	want := []*CheckRunAnnotation{{
		Path:            String("README.md"),
		BlobHRef:        String("https://github.com/octocat/Hello-World/blob/837db83be4137ca555d9a5598d0a1ea2987ecfee/README.md"),
		StartLine:       Int(2),
		EndLine:         Int(2),
		AnnotationLevel: String("warning"),
		Message:         String("Check your spelling for 'banaas'."),
		RawDetails:      String("Do you mean 'bananas' or 'banana'?"),
		Title:           String("Spell check"),
	}}

	if !reflect.DeepEqual(checkRunAnnotations, want) {
		t.Errorf("Checks.ListCheckRunAnnotations returned %+v, want %+v", checkRunAnnotations, want)
	}
}

func TestChecksService_UpdateCheckRun(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-runs/1", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PATCH")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		fmt.Fprint(w, `{
			"id": 1,
                        "name":"testUpdateCheckRun",
                        "head_sha":"deadbeef",
			"status": "completed",
			"conclusion": "neutral",
			"started_at": "2018-05-04T01:14:52Z",
			"completed_at": "2018-05-04T01:14:52Z",
                        "output":{"title": "Mighty test report", "summary":"There are 0 failures, 2 warnings and 1 notice", "text":"You may have misspelled some words."}}`)
	})
	startedAt, _ := time.Parse(time.RFC3339, "2018-05-04T01:14:52Z")
	updateCheckRunOpt := UpdateCheckRunOptions{
		HeadBranch:  String("master"),
		Name:        "testUpdateCheckRun",
		HeadSHA:     String("deadbeef"),
		Status:      String("completed"),
		CompletedAt: &Timestamp{startedAt},
		Output: &CheckRunOutput{
			Title:   String("Mighty test report"),
			Summary: String("There are 0 failures, 2 warnings and 1 notice"),
			Text:    String("You may have misspelled some words."),
		},
	}

	checkRun, _, err := client.Checks.UpdateCheckRun(context.Background(), "o", "r", 1, updateCheckRunOpt)
	if err != nil {
		t.Errorf("Checks.UpdateCheckRun return error: %v", err)
	}

	want := &CheckRun{
		ID:          Int64(1),
		Status:      String("completed"),
		StartedAt:   &Timestamp{startedAt},
		CompletedAt: &Timestamp{startedAt},
		Conclusion:  String("neutral"),
		HeadSHA:     String("deadbeef"),
		Name:        String("testUpdateCheckRun"),
		Output: &CheckRunOutput{
			Title:   String("Mighty test report"),
			Summary: String("There are 0 failures, 2 warnings and 1 notice"),
			Text:    String("You may have misspelled some words."),
		},
	}
	if !reflect.DeepEqual(checkRun, want) {
		t.Errorf("Checks.UpdateCheckRun return %+v, want %+v", checkRun, want)
	}
}

func TestChecksService_ListCheckRunsForRef(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/commits/master/check-runs", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		testFormValues(t, r, values{
			"check_name": "testing",
			"page":       "1",
			"status":     "completed",
			"filter":     "all",
		})
		fmt.Fprint(w, `{"total_count":1,
                                "check_runs": [{
                                    "id": 1,
                                    "head_sha": "deadbeef",
                                    "status": "completed",
                                    "conclusion": "neutral",
                                    "started_at": "2018-05-04T01:14:52Z",
                                    "completed_at": "2018-05-04T01:14:52Z"}]}`,
		)
	})

	opt := &ListCheckRunsOptions{
		CheckName:   String("testing"),
		Status:      String("completed"),
		Filter:      String("all"),
		ListOptions: ListOptions{Page: 1},
	}
	checkRuns, _, err := client.Checks.ListCheckRunsForRef(context.Background(), "o", "r", "master", opt)
	if err != nil {
		t.Errorf("Checks.ListCheckRunsForRef return error: %v", err)
	}
	startedAt, _ := time.Parse(time.RFC3339, "2018-05-04T01:14:52Z")
	want := &ListCheckRunsResults{
		Total: Int(1),
		CheckRuns: []*CheckRun{{
			ID:          Int64(1),
			Status:      String("completed"),
			StartedAt:   &Timestamp{startedAt},
			CompletedAt: &Timestamp{startedAt},
			Conclusion:  String("neutral"),
			HeadSHA:     String("deadbeef"),
		}},
	}

	if !reflect.DeepEqual(checkRuns, want) {
		t.Errorf("Checks.ListCheckRunsForRef returned %+v, want %+v", checkRuns, want)
	}
}

func TestChecksService_ListCheckRunsCheckSuite(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-suites/1/check-runs", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		testFormValues(t, r, values{
			"check_name": "testing",
			"page":       "1",
			"status":     "completed",
			"filter":     "all",
		})
		fmt.Fprint(w, `{"total_count":1,
                                "check_runs": [{
                                    "id": 1,
                                    "head_sha": "deadbeef",
                                    "status": "completed",
                                    "conclusion": "neutral",
                                    "started_at": "2018-05-04T01:14:52Z",
                                    "completed_at": "2018-05-04T01:14:52Z"}]}`,
		)
	})

	opt := &ListCheckRunsOptions{
		CheckName:   String("testing"),
		Status:      String("completed"),
		Filter:      String("all"),
		ListOptions: ListOptions{Page: 1},
	}
	checkRuns, _, err := client.Checks.ListCheckRunsCheckSuite(context.Background(), "o", "r", 1, opt)
	if err != nil {
		t.Errorf("Checks.ListCheckRunsCheckSuite return error: %v", err)
	}
	startedAt, _ := time.Parse(time.RFC3339, "2018-05-04T01:14:52Z")
	want := &ListCheckRunsResults{
		Total: Int(1),
		CheckRuns: []*CheckRun{{
			ID:          Int64(1),
			Status:      String("completed"),
			StartedAt:   &Timestamp{startedAt},
			CompletedAt: &Timestamp{startedAt},
			Conclusion:  String("neutral"),
			HeadSHA:     String("deadbeef"),
		}},
	}

	if !reflect.DeepEqual(checkRuns, want) {
		t.Errorf("Checks.ListCheckRunsCheckSuite returned %+v, want %+v", checkRuns, want)
	}
}

func TestChecksService_ListCheckSuiteForRef(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/commits/master/check-suites", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		testFormValues(t, r, values{
			"check_name": "testing",
			"page":       "1",
			"app_id":     "2",
		})
		fmt.Fprint(w, `{"total_count":1,
                                "check_suites": [{
                                    "id": 1,
                                    "head_sha": "deadbeef",
                                    "head_branch": "master",
                                    "status": "completed",
                                    "conclusion": "neutral",
                                    "before": "deadbeefb",
                                    "after": "deadbeefa"}]}`,
		)
	})

	opt := &ListCheckSuiteOptions{
		CheckName:   String("testing"),
		AppID:       Int(2),
		ListOptions: ListOptions{Page: 1},
	}
	checkSuites, _, err := client.Checks.ListCheckSuitesForRef(context.Background(), "o", "r", "master", opt)
	if err != nil {
		t.Errorf("Checks.ListCheckSuitesForRef return error: %v", err)
	}
	want := &ListCheckSuiteResults{
		Total: Int(1),
		CheckSuites: []*CheckSuite{{
			ID:         Int64(1),
			Status:     String("completed"),
			Conclusion: String("neutral"),
			HeadSHA:    String("deadbeef"),
			HeadBranch: String("master"),
			BeforeSHA:  String("deadbeefb"),
			AfterSHA:   String("deadbeefa"),
		}},
	}

	if !reflect.DeepEqual(checkSuites, want) {
		t.Errorf("Checks.ListCheckSuitesForRef returned %+v, want %+v", checkSuites, want)
	}
}

func TestChecksService_SetCheckSuitePreferences(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-suites/preferences", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "PATCH")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		fmt.Fprint(w, `{"preferences":{"auto_trigger_checks":[{"app_id": 2,"setting": false}]}}`)
	})
	p := &PreferenceList{
		AutoTriggerChecks: []*AutoTriggerCheck{{
			AppID:   Int64(2),
			Setting: Bool(false),
		}},
	}
	opt := CheckSuitePreferenceOptions{PreferenceList: p}
	prefResults, _, err := client.Checks.SetCheckSuitePreferences(context.Background(), "o", "r", opt)
	if err != nil {
		t.Errorf("Checks.SetCheckSuitePreferences return error: %v", err)
	}

	want := &CheckSuitePreferenceResults{
		Preferences: p,
	}

	if !reflect.DeepEqual(prefResults, want) {
		t.Errorf("Checks.SetCheckSuitePreferences return %+v, want %+v", prefResults, want)
	}
}

func TestChecksService_CreateCheckSuite(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-suites", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		fmt.Fprint(w, `{
			"id": 2,
                        "head_branch":"master",
                        "head_sha":"deadbeef",
			"status": "completed",
			"conclusion": "neutral",
                        "before": "deadbeefb",
                        "after": "deadbeefa"}`)
	})

	checkSuiteOpt := CreateCheckSuiteOptions{
		HeadSHA:    "deadbeef",
		HeadBranch: String("master"),
	}

	checkSuite, _, err := client.Checks.CreateCheckSuite(context.Background(), "o", "r", checkSuiteOpt)
	if err != nil {
		t.Errorf("Checks.CreateCheckSuite return error: %v", err)
	}

	want := &CheckSuite{
		ID:         Int64(2),
		Status:     String("completed"),
		HeadSHA:    String("deadbeef"),
		HeadBranch: String("master"),
		Conclusion: String("neutral"),
		BeforeSHA:  String("deadbeefb"),
		AfterSHA:   String("deadbeefa"),
	}
	if !reflect.DeepEqual(checkSuite, want) {
		t.Errorf("Checks.CreateCheckSuite return %+v, want %+v", checkSuite, want)
	}
}

func TestChecksService_ReRequestCheckSuite(t *testing.T) {
	client, mux, _, teardown := setup()
	defer teardown()

	mux.HandleFunc("/repos/o/r/check-suites/1/rerequest", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "POST")
		testHeader(t, r, "Accept", mediaTypeCheckRunsPreview)
		w.WriteHeader(http.StatusCreated)
	})
	resp, err := client.Checks.ReRequestCheckSuite(context.Background(), "o", "r", 1)
	if err != nil {
		t.Errorf("Checks.ReRequestCheckSuite return error: %v", err)
	}
	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("Checks.ReRequestCheckSuite = %v, want %v", got, want)
	}
}
