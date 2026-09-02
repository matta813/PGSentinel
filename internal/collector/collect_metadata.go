package collector

import "sort"

func changedSettings(before, after map[string]string) []string {
	keys := make([]string, 0)
	for key, value := range after {
		if old, ok := before[key]; ok && old != value {
			keys = append(keys, key+": "+old+" → "+value)
		}
	}
	sort.Strings(keys)
	if len(keys) > 50 {
		keys = keys[:50]
	}
	return keys
}
