package builder

import "github.com/buildpacks/pack/internal/dist"

const (
	OrderLabel = "io.buildpacks.buildpack.order"
)

type Metadata struct {
	Description string              `json:"description"`
	Buildpacks  []BuildpackMetadata `json:"buildpacks"`
	Groups      V1Order             `json:"groups"` // deprecated
	Stack       StackMetadata       `json:"stack"`
	Lifecycle   LifecycleMetadata   `json:"lifecycle"`
	CreatedBy   CreatorMetadata     `json:"createdBy"`
}

type CreatorMetadata struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type BuildpackMetadata struct {
	dist.BuildpackInfo
	Latest bool `json:"latest"` // deprecated
}

type LifecycleMetadata struct {
	LifecycleInfo
	API LifecycleAPI `json:"api"`
}

type StackMetadata struct {
	RunImage RunImageMetadata `json:"runImage" toml:"run-image"`
}

type RunImageMetadata struct {
	Image   string   `json:"image" toml:"image"`
	Mirrors []string `json:"mirrors" toml:"mirrors"`
}

func processMetadata(md *Metadata) {
	for i, bp := range md.Buildpacks {
		var matchingBps []dist.BuildpackInfo
		for _, bp2 := range md.Buildpacks {
			if bp.ID == bp2.ID {
				matchingBps = append(matchingBps, bp.BuildpackInfo)
			}
		}

		if len(matchingBps) == 1 {
			md.Buildpacks[i].Latest = true
		}
	}
}
