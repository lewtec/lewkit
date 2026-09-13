package path

import (
	"slices"
)

func names(ps []Path) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	slices.Sort(out)
	return out
}
