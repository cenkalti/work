package agent

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/cenkalti/work/internal/agent"
	"github.com/cenkalti/work/internal/slot"
	"github.com/spf13/cobra"
)

type listOpts struct {
	all     bool
	running bool
	idle    bool
}

func psCmd() *cobra.Command {
	var opts listOpts

	cmd := &cobra.Command{
		Use:   "ps",
		Short: "List agents across all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			recs, err := listAgentRecords(opts)
			if err != nil {
				return err
			}
			slots, _ := slot.Read()
			if isTerminal(os.Stdout) {
				return printAgentTable(os.Stdout, recs, slots)
			}
			for _, r := range recs {
				fmt.Println(agentHandle(r))
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&opts.all, "all", "a", false, "list all agents regardless of status")
	cmd.Flags().BoolVar(&opts.running, "running", false, "only show agents actively working (not idle)")
	cmd.Flags().BoolVar(&opts.idle, "idle", false, "only show agents with a running but idle session")

	return cmd
}

// listAgents returns identifiers in "project/branch" form for tab completion.
func listAgents(opts listOpts) ([]string, error) {
	recs, err := listAgentRecords(opts)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(recs))
	for _, r := range recs {
		out = append(out, agentHandle(r))
	}
	return out, nil
}

// agentHandle is the CLI handle used by `agent jump` and similar commands.
// It uses project/branch (or project for root agents) for back-compat.
func agentHandle(r *agent.Record) string {
	if r.Branch == "" {
		return r.Project
	}
	return r.Project + "/" + r.Branch
}

func listAgentRecords(opts listOpts) ([]*agent.Record, error) {
	all, err := agent.List()
	if err != nil {
		return nil, err
	}

	var sessionIDs map[string]struct{}
	if !opts.all {
		sessionIDs = agent.RunningSessionIDs()
	}

	out := make([]*agent.Record, 0, len(all))
	for _, r := range all {
		if !opts.all {
			if _, ok := sessionIDs[strings.ToLower(r.CurrentSessionID)]; !ok {
				continue
			}
		}
		if opts.running && r.Status != agent.StatusRunning && r.Status != agent.StatusToolRunning {
			continue
		}
		if opts.idle && r.Status != agent.StatusIdle && r.Status != agent.StatusAwaitingInput {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}

func printAgentTable(w *os.File, recs []*agent.Record, slots slot.Map) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	defer tw.Flush()
	fmt.Fprintln(tw, "SLOT\tPROJECT\tNAME\tSTATUS\tBRANCH")
	for _, r := range recs {
		slotStr := "-"
		for k, v := range slots {
			if v == r.ID {
				slotStr = fmt.Sprint(k)
				break
			}
		}
		branch := r.Branch
		if branch == "" {
			branch = "."
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", slotStr, r.Project, r.Name, r.Status, branch)
	}
	return nil
}
