package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/git"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// runningSessionIDs returns the set of session IDs from running claude processes.
func runningSessionIDs() (map[string]struct{}, error) {
	out, err := exec.Command("ps", "-eo", "args").Output()
	if err != nil {
		return nil, err
	}
	ids := make(map[string]struct{})
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || fields[0] != "claude" {
			continue
		}
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] == "--session-id" {
				ids[strings.ToLower(fields[i+1])] = struct{}{}
				break
			}
		}
	}
	return ids, nil
}

func run() error {
	running := flag.Bool("r", false, "only show agents with a running claude session")
	flag.Parse()

	var sessionIDs map[string]struct{}
	if *running {
		var err error
		sessionIDs, err = runningSessionIDs()
		if err != nil {
			return err
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	projectsDir := filepath.Join(home, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectPath := filepath.Join(projectsDir, entry.Name())
		worktrees, err := git.ListWorktrees(projectPath)
		if err != nil {
			continue
		}
		for _, wt := range worktrees {
			content, err := os.ReadFile(filepath.Join(wt, ".agent"))
			if err != nil {
				continue
			}
			uuid := strings.TrimSpace(string(content))

			if *running {
				if _, ok := sessionIDs[strings.ToLower(uuid)]; !ok {
					continue
				}
			}

			if wt == projectPath {
				fmt.Println(entry.Name())
				continue
			}
			wtRoot := filepath.Join(projectPath, ".work", "tree")
			wtRootResolved, err := filepath.EvalSymlinks(wtRoot)
			if err != nil {
				continue
			}
			prefix := wtRootResolved + string(filepath.Separator)
			if name, ok := strings.CutPrefix(wt, prefix); ok {
				fmt.Println(entry.Name() + "/" + name)
			}
		}
	}
	return nil
}
