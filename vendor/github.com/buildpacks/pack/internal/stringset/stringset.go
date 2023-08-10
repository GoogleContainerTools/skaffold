package stringset

// FromSlice converts the given slice to a set in the form of unique keys in a map.
// The value associated with each key should not be relied upon. A value is present
// in the set if its key is present in the map, regardless of the key's value.
func FromSlice(strings []string) map[string]interface{} {
	set := map[string]interface{}{}
	for _, s := range strings {
		set[s] = nil
	}
	return set
}

// Compare performs a set comparison between two slices. `extra` represents elements present in
// `strings1` but not `strings2`. `missing` represents elements present in `strings2` that are
// missing from `strings1`. `common` represents elements present in both slices. Since the input
// slices are treated as sets, duplicates will be removed in any outputs.
func Compare(strings1, strings2 []string) (extra []string, missing []string, common []string) {
	set1 := FromSlice(strings1)
	set2 := FromSlice(strings2)

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
