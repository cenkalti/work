package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cenkalti/work/internal/inbox"
	"github.com/spf13/cobra"
)

func inboxCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "inbox",
		Short: "Show pending agent notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			if isTerminal(os.Stdout) {
				return watchInbox(cmd.Context(), os.Stdout)
			}
			return printInbox(os.Stdout, false)
		},
	}
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func watchInbox(parent context.Context, w io.Writer) error {
	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Fprint(w, "\x1b[?1049h\x1b[?25l")
	defer fmt.Fprint(w, "\x1b[?25h\x1b[?1049l")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		fmt.Fprint(w, "\x1b[H\x1b[2J")
		if err := printInbox(w, true); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func printInbox(w io.Writer, hyperlinks bool) error {
	msgs, err := inbox.List()
	if err != nil {
		return err
	}
	for _, msg := range msgs {
		age := time.Since(msg.Timestamp).Truncate(time.Second)
		line := fmt.Sprintf("%-40s %s ago", msg.Name(), age)
		if hyperlinks {
			line = fmt.Sprintf("\x1b]8;;agent-jump://%s\x1b\\%s\x1b]8;;\x1b\\", msg.Name(), line)
		}
		fmt.Fprintln(w, line)
	}
	return nil
}
