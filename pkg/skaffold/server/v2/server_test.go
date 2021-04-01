package v2

import (
	"context"
	"testing"

	proto "github.com/GoogleContainerTools/skaffold/proto/v2"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

// Mock structs and functions
type mockData struct {
	Build  bool
	Sync   bool
	Deploy bool
}

var data mockData
var done = make(chan bool)

func mockBuildIntentCallback() {
	data.Build = true
	done <- true
}
func mockSyncIntentCallback() {
	data.Sync = true
	done <- true
}
func mockDeployIntentCallback() {
	data.Deploy = true
	done <- true
}

func TestServer_Execute(t *testing.T) {
	tests := []struct {
		description  string
		request      *proto.UserIntentRequest
		numCallBacks int
		expected     mockData
	}{
		{
			description: "build intent",
			request: &proto.UserIntentRequest{
				Intent: &proto.Intent{
					Build: true,
				},
			},
			numCallBacks: 1,
			expected: mockData{
				Build: true,
			},
		},
		{
			description: "sync intent",
			request: &proto.UserIntentRequest{
				Intent: &proto.Intent{
					Sync: true,
				},
			},
			numCallBacks: 1,
			expected: mockData{
				Sync: true,
			},
		},
		{
			description: "deploy intent",
			request: &proto.UserIntentRequest{
				Intent: &proto.Intent{
					Deploy: true,
				},
			},
			numCallBacks: 1,
			expected: mockData{
				Deploy: true,
			},
		},
		{
			description: "build and deploy intent",
			request: &proto.UserIntentRequest{
				Intent: &proto.Intent{
					Build:  true,
					Deploy: true,
				},
			},
			numCallBacks: 2,
			expected: mockData{
				Build:  true,
				Deploy: true,
			},
		},
		{
			description: "build, sync, and deploy intent",
			request: &proto.UserIntentRequest{
				Intent: &proto.Intent{
					Build:  true,
					Deploy: true,
					Sync:   true,
				},
			},
			numCallBacks: 3,
			expected: mockData{
				Build:  true,
				Sync:   true,
				Deploy: true,
			},
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.Override(&resetStateOnBuild, func() {})
			t.Override(&resetStateOnDeploy, func() {})
			data = mockData{}

			// Setup server with mock callback functions and run Execute()
			Srv = &Server{
				BuildIntentCallback:  mockBuildIntentCallback,
				SyncIntentCallback:   mockSyncIntentCallback,
				DeployIntentCallback: mockDeployIntentCallback,
			}
			_, err := Srv.Execute(context.Background(), test.request)
			if err != nil {
				t.Fail()
			}

			// Ensure callbacks finish updating data
			for i := 0; i < test.numCallBacks; i++ {
				<-done
			}

			t.CheckDeepEqual(test.expected, data)
		})
	}
}
