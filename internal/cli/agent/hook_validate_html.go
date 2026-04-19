package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type validateHTMLPayload struct {
	HookEventName string `json:"hook_event_name"`
	AgentType     string `json:"agent_type"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		FilePath string `json:"file_path"`
	} `json:"tool_input"`
}

func validateHTMLCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "validate-html",
		Short:  "PostToolUse hook that validates HTML written by the presentation subagent",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := io.ReadAll(os.Stdin)
			if err != nil || len(data) == 0 {
				return nil
			}
			var p validateHTMLPayload
			if err := json.Unmarshal(data, &p); err != nil {
				return nil
			}
			if p.HookEventName != "PostToolUse" ||
				p.AgentType != "presentation" ||
				(p.ToolName != "Write" && p.ToolName != "Edit") ||
				!strings.HasSuffix(p.ToolInput.FilePath, ".html") {
				return nil
			}

			report := validateHTMLFile(cmd.Context(), p.ToolInput.FilePath)
			if report.Valid {
				return nil
			}
			fmt.Print(formatReport(report))
			return nil
		},
	}
}
