package stack

import (
	"sort"

	"github.com/buildpacks/pack/internal/dist"
	"github.com/buildpacks/pack/internal/stringset"
)

// MergeCompatible determines the allowable set of stacks that a combination of buildpacks may run on, given each
// buildpack's set of stacks. Compatibility between the two sets of buildpack stacks is defined by the following rules:
//
//   1. The stack must be supported by both buildpacks. That is, any resulting stack ID must appear in both input sets.
//   2. For each supported stack ID, all required mixins for all buildpacks must be provided by the result. That is,
// 		mixins for the stack ID in both input sets are unioned.
//
// ---
//
// Examples:
//
// 	stacksA = [{ID: "stack1", mixins: ["build:mixinA", "mixinB", "run:mixinC"]}}]
// 	stacksB = [{ID: "stack1", mixins: ["build:mixinA", "run:mixinC"]}}]
// 	result = [{ID: "stack1", mixins: ["build:mixinA", "mixinB", "run:mixinC"]}}]
//
// 	stacksA = [{ID: "stack1", mixins: ["build:mixinA"]}}, {ID: "stack2", mixins: ["mixinA"]}}]
// 	stacksB = [{ID: "stack1", mixins: ["run:mixinC"]}}, {ID: "stack2", mixins: ["mixinA"]}}]
// 	result = [{ID: "stack1", mixins: ["build:mixinA", "run:mixinC"]}}, {ID: "stack2", mixins: ["mixinA"]}}]
//
// 	stacksA = [{ID: "stack1", mixins: ["build:mixinA"]}}, {ID: "stack2", mixins: ["mixinA"]}}]
// 	stacksB = [{ID: "stack2", mixins: ["mixinA", "run:mixinB"]}}]
// 	result = [{ID: "stack2", mixins: ["mixinA", "run:mixinB"]}}]
//
// 	stacksA = [{ID: "stack1", mixins: ["build:mixinA"]}}]
// 	stacksB = [{ID: "stack2", mixins: ["mixinA", "run:mixinB"]}}]
// 	result = []
//
func MergeCompatible(stacksA []dist.Stack, stacksB []dist.Stack) []dist.Stack {
	set := map[string][]string{}

	for _, s := range stacksA {
		set[s.ID] = s.Mixins
	}

	var results []dist.Stack
	for _, s := range stacksB {
		if stackMixins, ok := set[s.ID]; ok {
			mixinsSet := stringset.FromSlice(append(stackMixins, s.Mixins...))
			var mixins []string
			for s := range mixinsSet {
				mixins = append(mixins, s)
			}
			sort.Strings(mixins)

			results = append(results, dist.Stack{
				ID:     s.ID,
				Mixins: mixins,
			})
		}
	}

	return results
}
