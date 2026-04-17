package work

import (
	"context"

	"github.com/cenkalti/work/internal/location"
	"github.com/spf13/cobra"
)

type workContextKey struct{}

func detectLocation(cmd *cobra.Command) (*location.Location, error) {
	if wc, ok := cmd.Context().Value(workContextKey{}).(*location.Location); ok {
		return wc, nil
	}
	wc, err := location.Detect()
	if err != nil {
		return nil, err
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return wc, nil
}
