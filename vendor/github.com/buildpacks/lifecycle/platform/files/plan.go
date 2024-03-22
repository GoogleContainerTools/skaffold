package files

import "github.com/buildpacks/lifecycle/buildpack"

// Plan is written by the detector as plan.toml to record the application dependencies requested by buildpacks,
// and the image extensions or buildpacks that provide them.
// A subset of the plan is presented to each image extension (during the `generate` phase)
// or buildpack (during the `build` phase)
// with the entries that the module is expected to provide.
// The location of the file can be specified by providing `-plan <path>` to the lifecycle.
type Plan struct {
	Entries []BuildPlanEntry `toml:"entries"`
}

func (p Plan) Find(kind, id string) buildpack.Plan {
	var extension bool
	if kind == buildpack.KindExtension {
		extension = true
	}
	var out []buildpack.Require
	for _, entry := range p.Entries {
		for _, provider := range entry.Providers {
			if provider.ID == id && provider.Extension == extension {
				out = append(out, entry.Requires...)
				break
			}
		}
	}
	return buildpack.Plan{Entries: out}
}

// FIXME: ensure at least one claimed entry of each name is provided by the BP
func (p Plan) Filter(metRequires []string) Plan {
	var out []BuildPlanEntry
	for _, planEntry := range p.Entries {
		if !containsEntry(metRequires, planEntry) {
			out = append(out, planEntry)
		}
	}
	return Plan{Entries: out}
}

func containsEntry(metRequires []string, entry BuildPlanEntry) bool {
	for _, met := range metRequires {
		for _, planReq := range entry.Requires {
			if met == planReq.Name {
				return true
			}
		}
	}
	return false
}

type BuildPlanEntry struct {
	Providers []buildpack.GroupElement `toml:"providers"`
	Requires  []buildpack.Require      `toml:"requires"`
}

func (be BuildPlanEntry) NoOpt() BuildPlanEntry {
	var out []buildpack.GroupElement
	for _, p := range be.Providers {
		out = append(out, p.NoOpt().NoAPI().NoHomepage())
	}
	be.Providers = out
	return be
}
