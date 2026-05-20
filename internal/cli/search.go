package cli

import (
	"fmt"
	"strings"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <pattern>",
		Short: "Search clause bodies for a substring (case-insensitive).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pattern := strings.ToLower(args[0])
			w := cmd.OutOrStdout()

			unionDir, err := paths.UnionDir()
			if err != nil {
				return err
			}
			stores, err := store.ListStores(unionDir)
			if err != nil {
				return err
			}

			var hits int
			for _, name := range stores {
				s, err := store.OpenNamed(unionDir, name)
				if err != nil {
					return err
				}
				clauses, err := s.List("")
				if err != nil {
					return err
				}
				for _, p := range clauses {
					body, err := s.Get(p)
					if err != nil {
						fmt.Fprintf(w, "WARN: could not read %s:%s: %v\n", name, p, err)
						continue
					}
					if strings.Contains(strings.ToLower(string(body)), pattern) {
						fmt.Fprintf(w, "%s:%s\n", name, p)
						hits++
					}
				}
			}
			if hits == 0 {
				fmt.Fprintln(w, "no matches")
			}
			return nil
		},
	}
}
