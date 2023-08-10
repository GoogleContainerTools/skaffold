package str

// compare performs a set comparison between two slices. `extra` represents elements present in
// `strings1` but not `strings2`. `missing` represents elements present in `strings2` that are
// missing from `strings1`. `common` represents elements present in both slices. Since the input
// slices are treated as sets, duplicates will be removed in any outputs.
// from: https://github.com/buildpacks/pack/blob/main/internal/stringset/stringset.go
func Compare(strings1, strings2 []string) (extra []string, missing []string, common []string) {
	set1 := map[string]struct{}{}
	set2 := map[string]struct{}{}
	for _, s := range strings1 {
		set1[s] = struct{}{}
	}
	for _, s := range strings2 {
		set2[s] = struct{}{}
	}

	for s := range set1 {
		if _, ok := set2[s]; !ok {
			extra = append(extra, s)
			continue
		}
		common = append(common, s)
	}

	for s := range set2 {
		if _, ok := set1[s]; !ok {
			missing = append(missing, s)
		}
	}

	return extra, missing, common
}
