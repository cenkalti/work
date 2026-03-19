package work

import (
	"context"
	"fmt"
	"os"

	"github.com/cenkalti/work/internal/location"
	"github.com/spf13/cobra"
)

type workContextKey struct{}

func persistWorkContext(cmd *cobra.Command, args []string) error {
	wc, err := location.Detect()
	if err != nil {
		return err
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return nil
}

func detectLocation(cmd *cobra.Command) *location.Location {
	if wc, ok := cmd.Context().Value(workContextKey{}).(*location.Location); ok {
		return wc
	}
	wc, err := location.Detect()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	cmd.SetContext(context.WithValue(cmd.Context(), workContextKey{}, wc))
	return wc
}
