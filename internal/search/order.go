package search

import "sort"

func sortedKeys(set map[string]bool) []string {
	values := make([]string, 0, len(set))
	for key := range set {
		values = append(values, key)
	}
	sort.Strings(values)
	return values
}
