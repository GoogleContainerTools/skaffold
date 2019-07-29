package runner

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"

	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGeneratePipeline(t *testing.T) {
	var tests = []struct {
		description string
		tasks       []*tekton.Task
		shouldErr   bool
	}{
		{
			description: "successful tekton pipeline generation",
			tasks: []*tekton.Task{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-task",
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating tekton pipeline",
			tasks:       []*tekton.Task{},
			shouldErr:   true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := generatePipeline(test.tasks)

			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateBuildTask(t *testing.T) {
	var tests = []struct {
		description string
		buildConfig latest.BuildConfig
		shouldErr   bool
	}{
		{
			description: "successfully generate build task",
			buildConfig: latest.BuildConfig{
				Artifacts: []*latest.Artifact{
					{
						ImageName: "testArtifact",
					},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating build task",
			buildConfig: latest.BuildConfig{
				Artifacts: []*latest.Artifact{},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := generateBuildTask(test.buildConfig)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateDeployTask(t *testing.T) {
	var tests = []struct {
		description  string
		deployConfig latest.DeployConfig
		shouldErr    bool
	}{
		{
			description: "successfully generate deploy task",
			deployConfig: latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy: &latest.HelmDeploy{},
				},
			},
			shouldErr: false,
		},
		{
			description: "fail generating deploy task",
			deployConfig: latest.DeployConfig{
				DeployType: latest.DeployType{
					HelmDeploy:      nil,
					KubectlDeploy:   nil,
					KustomizeDeploy: nil,
				},
			},
			shouldErr: true,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			_, err := generateDeployTask(test.deployConfig)
			t.CheckError(test.shouldErr, err)
		})
	}
}

func TestGenerateProfile(t *testing.T) {
	var tests = []struct {
		description     string
		skaffoldConfig  *latest.SkaffoldConfig
		expectedProfile *latest.Profile
		responses       []string
		shouldErr       bool
	}{
		{
			description: "successful profile generation",
			skaffoldConfig: &latest.SkaffoldConfig{
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "test",
							},
						},
					},
				},
			},
			expectedProfile: &latest.Profile{
				Name: "oncluster",
				Pipeline: latest.Pipeline{
					Build: latest.BuildConfig{
						Artifacts: []*latest.Artifact{
							{
								ImageName: "test-pipeline",
							},
						},
						BuildType: latest.BuildType{
							Cluster: &latest.ClusterDetails{
								PullSecretName: "kaniko-secret",
							},
						},
					},
				},
			},
			shouldErr: false,
		},
	}

	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			profile, err := generateProfile(test.skaffoldConfig)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expectedProfile, profile)
		})
	}
}
