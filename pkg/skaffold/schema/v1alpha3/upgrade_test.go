package v1alpha3

import (
	"testing"
	v1alpha4 "github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
)

func TestBuildUpgrade(t *testing.T) {
	old := `apiVersion: skaffold/v1alpha3
kind: Config
build:
  local:	
    skipPush: false
profiles:
  - name: testEnv1
    build:
      local:
        skipPush: true
  - name: testEnv2
    build:
      local:
        skipPush: false
`
	pipeline := NewSkaffoldPipeline()
	err := pipeline.Parse([]byte(old), true)
	if err != nil {
		t.Errorf("unexpected error during parsing old config: %v", err)
	}

	upgraded, err := pipeline.Upgrade()
	if err != nil {
		t.Errorf("unexpected error during upgrade: %v", err)
	}

	upgradedPipeline := upgraded.(*v1alpha4.SkaffoldPipeline)

	if upgradedPipeline.Build.LocalBuild == nil {
		t.Errorf("expected build.local to be not nil")
	}
	if upgradedPipeline.Build.LocalBuild.Push != nil && *upgradedPipeline.Build.LocalBuild.Push != true {
		t.Errorf("expected build.local.push to be true but it was: %v", *upgradedPipeline.Build.LocalBuild.Push)
	}

	if upgradedPipeline.Profiles[0].Build.LocalBuild == nil {
		t.Errorf("expected profiles[0].build.local to be not nil")
	}
	if upgradedPipeline.Profiles[0].Build.LocalBuild.Push != nil && *upgradedPipeline.Profiles[0].Build.LocalBuild.Push != false {
		t.Errorf("expected profiles[0].build.local.push to be false but it was: %v", *upgradedPipeline.Build.LocalBuild.Push)
	}

	if upgradedPipeline.Profiles[1].Build.LocalBuild == nil {
		t.Errorf("expected profiles[1].build.local to be not nil")
	}
	if upgradedPipeline.Profiles[1].Build.LocalBuild.Push != nil && *upgradedPipeline.Profiles[1].Build.LocalBuild.Push != true {
		t.Errorf("expected profiles[1].build.local.push to be true but it was: %v", *upgradedPipeline.Build.LocalBuild.Push)
	}
}
