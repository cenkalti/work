package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cenkalti/work/internal/paths"
	taskpkg "github.com/cenkalti/work/internal/task"
	"github.com/spf13/cobra"
)

func migrateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Convert task files from JSON to YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			return runMigrate(paths.LocalTasksDir(cwd))
		},
	}
}

func runMigrate(tasksDir string) error {
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return fmt.Errorf("reading tasks dir: %w", err)
	}

	count := 0
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(tasksDir, e.Name()))
		if err != nil {
			return err
		}
		var t taskpkg.Task
		if err := json.Unmarshal(data, &t); err != nil {
			return fmt.Errorf("parsing %s: %w", e.Name(), err)
		}
		if err := t.WriteToFile(tasksDir); err != nil {
			return fmt.Errorf("writing %s: %w", t.ID, err)
		}
		if err := os.Remove(filepath.Join(tasksDir, e.Name())); err != nil {
			return fmt.Errorf("removing %s: %w", e.Name(), err)
		}
		fmt.Printf("Migrated %s\n", t.ID)
		count++
	}

	if count == 0 {
		fmt.Println("No JSON task files found")
	} else {
		fmt.Printf("Migrated %d task(s) to YAML\n", count)
	}
	return nil
}
