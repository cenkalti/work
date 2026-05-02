package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/work/internal/wezterm"
	"github.com/spf13/cobra"
)

func dashFocusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dash-focus",
		Short: "Focus the agent dashboard window (spawn one if missing)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDashFocus()
		},
	}
}

func runDashFocus() error {
	if id, ok := readDashPaneID(); ok {
		if _, found := wezterm.FindPaneByID(id); found {
			return wezterm.ActivatePane(id)
		}
	}

	if _, err := wezterm.ListPanes(); err != nil {
		_ = exec.Command("open", "-a", "WezTerm").Run()
		time.Sleep(500 * time.Millisecond)
		if _, err := wezterm.ListPanes(); err != nil {
			return err
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating agent binary: %w", err)
	}
	paneID, err := wezterm.SpawnNewWindow("", exe, "dash")
	if err != nil {
		return err
	}
	return wezterm.ActivatePane(paneID)
}

func readDashPaneID() (int, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return 0, false
	}
	data, err := os.ReadFile(filepath.Join(home, ".work", "dash.pane"))
	if err != nil {
		return 0, false
	}
	id, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, false
	}
	return id, true
}
