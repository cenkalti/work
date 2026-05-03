package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/work/internal/domain"
	"github.com/cenkalti/work/internal/session"
	"github.com/cenkalti/work/internal/wezterm"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// createAgentWorktree creates a worktree for the given <project>/<branch> id
// using the same logic as `work mk`. Returns the resulting worktree path.
func createAgentWorktree(id string) (string, error) {
	project, branch, _ := strings.Cut(id, "/")
	if branch == "" {
		return "", fmt.Errorf("unknown agent: %s", id)
	}
	projectsDir, err := domain.ProjectsDir()
	if err != nil {
		return "", err
	}
	rootRepo := filepath.Join(projectsDir, project)
	if _, err := os.Stat(rootRepo); err != nil {
		return "", fmt.Errorf("unknown project: %s", project)
	}
	wt := domain.Worktree{RepoPath: rootRepo, Name: branch}
	gitBranch := os.Getenv("WORK_BRANCH_PREFIX") + branch
	wtPath, err := session.Create(wt, gitBranch, false)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(wtPath); err == nil {
		wtPath = resolved
	}
	return wtPath, nil
}

func runCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run [<project>[/<branch>]]",
		Short: "Start or resume a claude session",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names, _ := listAgents(listOpts{all: true})
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			claudeBin, err := exec.LookPath("claude")
			if err != nil {
				return err
			}

			if len(args) == 1 {
				path, err := resolveAgentPath(args[0])
				if err != nil {
					path, err = createAgentWorktree(args[0])
					if err != nil {
						return err
					}
				}
				if err := os.Chdir(path); err != nil {
					return err
				}
			}

			rec, err := loadOrCreateAgent()
			if err != nil {
				return err
			}

			rec.PaneID = os.Getenv("WEZTERM_PANE")
			if id, err := strconv.Atoi(rec.PaneID); err == nil {
				if p, ok := wezterm.FindPaneByID(id); ok {
					rec.TTYName = p.TTYName
				}
			}
			rec.UpdatedAt = time.Now().UTC()

			var claudeArgs []string
			if rec.SessionID != "" && claudeSessionExists(rec.WorktreePath, rec.SessionID) {
				claudeArgs = []string{"claude", "--resume", rec.SessionID}
			} else {
				rec.SessionID = uuid.New().String()
				claudeArgs = []string{"claude", "--session-id", rec.SessionID}
			}

			if err := rec.Save(); err != nil {
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
func loadOrCreateAgent() (*domain.Agent, error) {
	repo, wt, err := domain.Detect()
	if err != nil {
		return nil, err
	}

	var worktreePath, branchName string
	if wt == nil {
		worktreePath = repo.Path
		if _, err := repo.EnsureRootWorkspace(); err != nil {
			return nil, err
		}
	} else {
		worktreePath = wt.Path()
		branchName = wt.Name
	}
	resolved, err := filepath.EvalSymlinks(worktreePath)
	if err == nil {
		worktreePath = resolved
	}

	rec, err := domain.FindAgentByWorktree(worktreePath)
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

	taskID := branchName
	if taskID == "" {
		taskID = repo.ProjectName()
	} else {
		taskID = domain.BranchID(branchName)
	}

	return &domain.Agent{
		UUID:         id.String(),
		Name:         taskID,
		RepoPath:     repo.Path,
		WorktreeName: branchName,
		WorktreePath: worktreePath,
		Status:       domain.StatusIdle,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}
