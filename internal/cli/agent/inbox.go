package agent

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/cenkalti/work/internal/agent"
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
	running := agent.RunningSessionIDs()
	live := msgs[:0]
	for _, msg := range msgs {
		if _, ok := running[strings.ToLower(msg.SessionID)]; !ok {
			_ = inbox.Delete(msg.SessionID)
			continue
		}
		live = append(live, msg)
	}
	msgs = live
	nameWidth, ageWidth := 0, 0
	ages := make([]string, len(msgs))
	for i, msg := range msgs {
		ages[i] = time.Since(msg.Timestamp).Truncate(time.Second).String()
		if n := len(msg.Name()); n > nameWidth {
			nameWidth = n
		}
		if a := len(ages[i]); a > ageWidth {
			ageWidth = a
		}
	}
	for i, msg := range msgs {
		name := msg.Name()
		display := name
		if hyperlinks {
			display = fmt.Sprintf("\x1b]8;;agent-jump://%s\x1b\\%s\x1b]8;;\x1b\\", name, name)
		}
		pad := strings.Repeat(" ", nameWidth-len(name))
		fmt.Fprintf(w, "%s%s  %*s ago\n", display, pad, ageWidth, ages[i])
	}
	return nil
}
