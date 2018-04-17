package misc

import (
	"sort"
)

// Remove duplicate elements from a sorted list of strings.
func SortedDedup(names []string) []string {
	if len(names) < 2 {
		return names
	}

	j := 0
	for i := 1; i < len(names); i++ {
		if names[j] == names[i] {
			continue
		}
		j++
		names[j] = names[i]
	}
	return names[:j+1]
}

// Checks for the presence of a string in a sorted list of strings.
func SortedHasString(list []string, tofind string) bool {
	index := sort.SearchStrings(list, tofind)
	if index < len(list) && list[index] == tofind {
		return true
	}
	return false
}

func ArrayStringIndex(list []string, tofind string) int {
	for i := range list {
		if list[i] == tofind {
			return i
		}
	}
	return -1
}
