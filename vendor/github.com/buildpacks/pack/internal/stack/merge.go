package stack

import (
	"sort"

	"github.com/buildpacks/pack/internal/stringset"
	"github.com/buildpacks/pack/pkg/dist"
)

const WildcardStack = "*"

// MergeCompatible determines the allowable set of stacks that a combination of buildpacks may run on, given each
// buildpack's set of stacks. Compatibility between the two sets of buildpack stacks is defined by the following rules:
//
//   1. The stack must be supported by both buildpacks. That is, any resulting stack ID must appear in both input sets.
//   2. For each supported stack ID, all required mixins for all buildpacks must be provided by the result. That is,
// 		mixins for the stack ID in both input sets are unioned.
//   3. If there is a wildcard stack in either of the stack list, the stack list not having the wild card stack is returned.
//   4. If both the stack lists contain a wildcard stack, a list containing just the wildcard stack is returned.
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
// 	stacksA = [{ID: "*"}, {ID: "stack1", mixins: ["build:mixinC"]}]
// 	stacksB = [{ID: "stack1", mixins: ["build:mixinA"]}, {ID: "stack2", mixins: ["mixinA", "run:mixinB"]}]
// 	result = [{ID: "stack1", mixins: ["build:mixinA"]}, {ID: "stack2", mixins: ["mixinA", "run:mixinB"]}]
//
// 	stacksA = [{ID: "stack1", mixins: ["build:mixinA"]}, {ID: "stack2", mixins: ["mixinA", "run:mixinB"]}]
// 	stacksB = [{ID: "*"}, {ID: "stack1", mixins: ["build:mixinC"]}]
// 	result = [{ID: "stack1", mixins: ["build:mixinA"]}, {ID: "stack2", mixins: ["mixinA", "run:mixinB"]}]
//
// 	stacksA = [{ID: "*"}, {ID: "stack1", mixins: ["build:mixinA"]}, {ID: "stack2", mixins: ["mixinA", "run:mixinB"]}]
// 	stacksB = [{ID: "*"}, {ID: "stack1", mixins: ["build:mixinC"]}]
// 	result = [{ID: "*"}]
//
func MergeCompatible(stacksA []dist.Stack, stacksB []dist.Stack) []dist.Stack {
	set := map[string][]string{}
	AHasWildcardStack, BHasWildcardStack := false, false

	for _, s := range stacksA {
		set[s.ID] = s.Mixins
		if s.ID == WildcardStack {
			AHasWildcardStack = true
		}
	}

	for _, s := range stacksB {
		if s.ID == WildcardStack {
			BHasWildcardStack = true
		}
	}

	if AHasWildcardStack && BHasWildcardStack {
		return []dist.Stack{{ID: WildcardStack}}
	}

	if AHasWildcardStack {
		return stacksB
	}

	if BHasWildcardStack {
		return stacksA
	}

	var results []dist.Stack

	for _, s := range stacksB {
		if stackMixins, ok := set[s.ID]; ok {
			mixinsSet := stringset.FromSlice(append(stackMixins, s.Mixins...))
			var mixins []string
			for m := range mixinsSet {
				mixins = append(mixins, m)
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
