package agent

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/location"
	"github.com/cenkalti/work/internal/paths"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start or resume a claude session",
		RunE: func(cmd *cobra.Command, args []string) error {
			claudeBin, err := exec.LookPath("claude")
			if err != nil {
				return err
			}

			rec, err := loadOrCreateAgent()
			if err != nil {
				return err
			}

			rec.PaneID = os.Getenv("WEZTERM_PANE")
			rec.UpdatedAt = time.Now().UTC()

			var claudeArgs []string
			if rec.CurrentSessionID != "" && claudeSessionExists(rec.WorktreePath, rec.CurrentSessionID) {
				claudeArgs = []string{"claude", "--resume", rec.CurrentSessionID}
			} else {
				rec.CurrentSessionID = uuid.New().String()
				claudeArgs = []string{"claude", "--session-id", rec.CurrentSessionID}
			}

			if err := agent.Write(rec); err != nil {
				return err
			}
			return syscall.Exec(claudeBin, claudeArgs, os.Environ())
		},
	}
}

// claudeSessionExists reports whether a Claude Code conversation file exists
// for the given session in the project directory matching worktreePath.
// Claude stores conversations at ~/.claude/projects/<encoded-cwd>/<session-id>.jsonl
// where the encoding replaces "/" with "-".
func claudeSessionExists(worktreePath, sessionID string) bool {
	if worktreePath == "" || sessionID == "" {
		return false
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	encoded := strings.ReplaceAll(worktreePath, "/", "-")
	p := filepath.Join(home, ".claude", "projects", encoded, sessionID+".jsonl")
	_, err = os.Stat(p)
	return err == nil
}

// loadOrCreateAgent returns the agent record for the current worktree, creating
// one if needed.
func loadOrCreateAgent() (*agent.Record, error) {
	loc, err := location.Detect()
	if err != nil {
		return nil, err
	}

	var worktreePath string
	if loc.IsRoot() {
		worktreePath = loc.RootRepo
	} else {
		worktreePath = paths.Worktree(loc.RootRepo, loc.Branch)
	}
	resolved, err := filepath.EvalSymlinks(worktreePath)
	if err == nil {
		worktreePath = resolved
	}

	rec, err := agent.FindByWorktree(worktreePath)
	if err == nil {
		return rec, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()

	project := filepath.Base(loc.RootRepo)
	taskID := loc.Branch
	if taskID == "" {
		taskID = project
	} else {
		taskID = paths.BranchID(loc.Branch)
	}

	return &agent.Record{
		ID:           id.String(),
		Name:         taskID,
		Project:      project,
		ProjectRoot:  loc.RootRepo,
		TaskID:       taskID,
		Branch:       loc.Branch,
		WorktreePath: worktreePath,
		Status:       agent.StatusIdle,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
