package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/shop"
	"github.com/spf13/cobra"
)

func newOrganizeCmd() *cobra.Command {
	var contract string
	cmd := &cobra.Command{
		Use:   "organize [dir]",
		Short: "Register a directory as an organized shop (default: .).",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := "."
			if len(args) == 1 {
				dir = args[0]
			}
			abs, err := resolveDir(dir)
			if err != nil {
				return err
			}
			if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
				return fmt.Errorf("not a directory: %s", abs)
			}
			shopsPath, err := paths.ShopsFile()
			if err != nil {
				return err
			}
			r, err := shop.LoadRegistry(shopsPath)
			if err != nil {
				return err
			}
			if err := r.Add(abs, contract); err != nil {
				return err
			}
			if err := r.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "organized shop %s (contract: %s)\n", abs, effectiveContract(contract))
			return nil
		},
	}
	cmd.Flags().StringVar(&contract, "contract", "AGENTS.md", "contract filename within the shop")
	return cmd
}

func effectiveContract(c string) string {
	if c == "" {
		return "AGENTS.md"
	}
	return c
}

// resolveDir returns the absolute, symlink-resolved path for dir so that
// shops recorded on macOS /tmp work when the cwd reports /private/tmp (and
// vice versa).
func resolveDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(abs); err == nil {
		return real, nil
	}
	return abs, nil
}
