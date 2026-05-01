package agent

import (
	"errors"
	"fmt"
	"io/fs"
	"time"

	"github.com/cenkalti/work/internal/agent"
	"github.com/spf13/cobra"
)

func mvCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mv <name> <new-name>",
		Short: "Rename an agent",
		Args:  cobra.ExactArgs(2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names, _ := agentNames()
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName, newName := args[0], args[1]
			if oldName == newName {
				return nil
			}
			rec, err := findAgentByName(oldName)
			if err != nil {
				return err
			}
			if other, err := findAgentByName(newName); err == nil && other.ID != rec.ID {
				return fmt.Errorf("name %q is already in use", newName)
			} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			rec.Name = newName
			rec.UpdatedAt = time.Now().UTC()
			if err := agent.Write(rec); err != nil {
				return err
			}
			fmt.Printf("renamed %s -> %s\n", oldName, newName)
			return nil
		},
	}
}

// findAgentByName returns the unique agent with Name == name.
// Returns fs.ErrNotExist if no match. Errors if multiple match.
func findAgentByName(name string) (*agent.Record, error) {
	all, err := agent.List()
	if err != nil {
		return nil, err
	}
	var matches []*agent.Record
	for _, r := range all {
		if r.Name == name {
			matches = append(matches, r)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fs.ErrNotExist
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("name %q matches multiple agents (%d); use a unique name", name, len(matches))
	}
}

// agentNames returns the set of agent names for tab completion.
func agentNames() ([]string, error) {
	all, err := agent.List()
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(all))
	for _, r := range all {
		out = append(out, r.Name)
	}
	return out, nil
}
