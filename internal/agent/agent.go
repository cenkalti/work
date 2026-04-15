package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const FileName = ".agent"

const (
	StatusRunning = "running"
	StatusIdle    = "idle"
	StatusEnded   = "ended"
)

type State struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func FilePath(dir string) string {
	return filepath.Join(dir, FileName)
}

func Read(dir string) (*State, error) {
	data, err := os.ReadFile(FilePath(dir))
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func Write(dir string, s *State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(FilePath(dir), data, 0644)
}

// IsSessionRunning checks if a claude process with the given session ID is running.
func IsSessionRunning(sessionID string) bool {
	out, err := exec.Command("ps", "-eo", "args").Output()
	if err != nil {
		return false
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "claude" {
			continue
		}
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] == "--session-id" && strings.EqualFold(fields[i+1], sessionID) {
				return true
			}
		}
	}
	return false
}

// RunningSessionIDs returns the set of session IDs from running claude processes.
func RunningSessionIDs() map[string]struct{} {
	out, err := exec.Command("ps", "-eo", "args").Output()
	if err != nil {
		return nil
	}
	ids := make(map[string]struct{})
	for line := range strings.SplitSeq(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 || fields[0] != "claude" {
			continue
		}
		for i := 1; i < len(fields)-1; i++ {
			if fields[i] == "--session-id" {
				ids[strings.ToLower(fields[i+1])] = struct{}{}
				break
			}
		}
	}
	return ids
}

// HookInput is the common JSON structure received on stdin from Claude Code hooks.
type HookInput struct {
	SessionID string `json:"session_id"`
}

// ReadHookInput reads and parses the hook JSON from stdin.
func ReadHookInput() (*HookInput, error) {
	data, err := os.ReadFile("/dev/stdin")
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	var input HookInput
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("parsing hook input: %w", err)
	}
	if input.SessionID == "" {
		return nil, fmt.Errorf("no session_id in hook input")
	}
	return &input, nil
}
