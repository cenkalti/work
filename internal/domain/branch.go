package domain

import "strings"

// Branch is a git branch in a Repo. Name uses dots to encode hierarchy:
// "a.b.c" means task c, child of b, child of root task a.
type Branch struct {
	RepoPath string
	Name     string
}

// IsRoot reports whether the branch is a root task (no dots).
func (b Branch) IsRoot() bool {
	return !strings.Contains(b.Name, ".")
}

// Parent returns the branch with the last segment stripped. Returns the
// zero Branch (Name == "") for root branches.
func (b Branch) Parent() Branch {
	i := strings.LastIndex(b.Name, ".")
	if i < 0 {
		return Branch{RepoPath: b.RepoPath}
	}
	return Branch{RepoPath: b.RepoPath, Name: b.Name[:i]}
}

// ID returns the last segment after the final dot. For root branches it
// returns the full name.
func (b Branch) ID() string {
	return BranchID(b.Name)
}

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
