package cli

import (
	"strings"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func completeClausePath(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	unionDir, err := paths.UnionDir()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	stores, err := store.ListStores(unionDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, name := range stores {
		if !strings.HasPrefix(toComplete, name+":") && strings.Contains(toComplete, ":") {
			continue
		}
		s, err := store.OpenNamed(unionDir, name)
		if err != nil {
			continue
		}
		prefix := ""
		if i := strings.IndexByte(toComplete, ':'); i >= 0 {
			prefix = toComplete[i+1:]
		}
		clauses, err := s.List(prefix)
		if err != nil {
			continue
		}
		for _, c := range clauses {
			completions = append(completions, name+":"+c)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
