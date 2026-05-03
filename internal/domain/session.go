package domain

import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Session is a Claude Code session, identified by the session ID assigned
// by Claude Code.
type Session struct {
	ID string
}

// IsRunning reports whether a claude process is currently running with this
// session ID (either via --session-id or --resume).
func (s Session) IsRunning() bool {
	if s.ID == "" {
		return false
	}
	_, ok := RunningSessionIDs()[strings.ToLower(s.ID)]
	return ok
}

// RunningSessionIDs returns the set of session IDs from running claude
// processes. Keys are lower-cased session IDs.
func RunningSessionIDs() map[string]struct{} {
	out, err := exec.Command("ps", "-eo", "args").Output()
	if err != nil {
		return nil
	}
	ids := make(map[string]struct{})
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || filepath.Base(fields[0]) != "claude" {
			continue
		}
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] == "--resume" || fields[i] == "--session-id" {
				ids[strings.ToLower(fields[i+1])] = struct{}{}
				break
			}
		}
	}
	return ids
}
