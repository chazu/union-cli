package cli

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/chazu/union/internal/qpath"
	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	return &cobra.Command{
		Use:               "log <store:path>",
		Short:             "Show git log for a clause.",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeClausePath,
		RunE: func(cmd *cobra.Command, args []string) error {
			q, err := qpath.Parse(args[0])
			if err != nil {
				return err
			}
			s, err := openStoreFor(q)
			if err != nil {
				return err
			}
			relPath := filepath.Join("clauses", filepath.FromSlash(q.Path)+".md")
			gitCmd := exec.Command("git", "log", "--follow", "-p", "--", relPath)
			gitCmd.Dir = s.Root()
			gitCmd.Stdout = os.Stdout
			gitCmd.Stderr = os.Stderr
			return gitCmd.Run()
		},
	}
}
