package domain

import "strings"

// ParentBranchName returns everything before the last dot. Returns "" for
// root branches.
func ParentBranchName(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[:i]
	}
	return ""
}

// BranchID returns the last segment after the final dot. Returns the full
// name for root branches.
func BranchID(name string) string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		return name[i+1:]
	}
	return name
}
