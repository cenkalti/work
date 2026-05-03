package work

import (
	"context"

	"github.com/cenkalti/work/internal/domain"
	"github.com/spf13/cobra"
)

type workContextKey struct{}

type workCtx struct {
	Repo     domain.Repo
	Worktree *domain.Worktree
}

func detectLocation(cmd *cobra.Command) (*workCtx, error) {
	if wc, ok := cmd.Context().Value(workContextKey{}).(*workCtx); ok {
		return wc, nil
	}
	repo, wt, err := domain.Detect()
	if err != nil {
		return nil, err
	}
	wc := &workCtx{Repo: repo, Worktree: wt}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return wc, nil
}
