package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chazu/union/internal/harness"
	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import existing configurations into union clauses.",
	}
	cmd.AddCommand(newImportHooksCmd())
	return cmd
}

func newImportHooksCmd() *cobra.Command {
	var (
		harnessName string
		storeName   string
		prefix      string
	)
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Import hooks from native harness configs into union clauses.",
		Long: `Reads hooks from detected harness config files and creates union
hook clauses for each one. Imported clauses are stored but not automatically
ratified — use 'union ratify' to add them to the contract.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runImportHooks(cmd, harnessName, storeName, prefix)
		},
	}
	cmd.Flags().StringVar(&harnessName, "harness", "", "Import from this harness only")
	cmd.Flags().StringVar(&storeName, "store", "default", "Target store for imported clauses")
	cmd.Flags().StringVar(&prefix, "prefix", "hooks", "Clause path prefix")
	return cmd
}

func runImportHooks(cmd *cobra.Command, filterHarness, storeName, prefix string) error {
	s, _, err := currentShop()
	if err != nil {
		return err
	}
	w := cmd.OutOrStdout()

	adapters, err := harness.ResolveHarnesses(s.Dir)
	if err != nil {
		return err
	}
	if filterHarness != "" {
		var filtered []harness.Adapter
		for _, a := range adapters {
			if a.Name() == filterHarness {
				filtered = append(filtered, a)
			}
		}
		adapters = filtered
	}
	if len(adapters) == 0 {
		return fmt.Errorf("no harnesses found to import from")
	}

	// Open target store.
	unionDir, err := paths.UnionDir()
	if err != nil {
		return err
	}
	st, err := store.OpenNamed(unionDir, storeName)
	if err != nil {
		return fmt.Errorf("open store %q: %w", storeName, err)
	}

	imported := 0
	for _, adapter := range adapters {
		settingsPath := filepath.Join(s.Dir, adapter.SettingsPath())
		config, err := os.ReadFile(settingsPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", settingsPath, err)
		}

		hooks, err := adapter.Import(config)
		if err != nil {
			return fmt.Errorf("import from %s: %w", adapter.Name(), err)
		}
		if len(hooks) == 0 {
			continue
		}

		fmt.Fprintf(w, "From %s (%s):\n", adapter.Name(), adapter.SettingsPath())
		for _, h := range hooks {
			clausePath := filepath.Join(prefix, slugify(h.Event, h.Matcher))
			body := renderHookClause(h, adapter.Name())

			if st.Has(clausePath) {
				fmt.Fprintf(w, "  ~ %s:%s (exists, skipped)\n", storeName, clausePath)
				continue
			}

			msg := fmt.Sprintf("import hook from %s: %s", adapter.Name(), clausePath)
			if err := st.Put(clausePath, []byte(body), msg); err != nil {
				return fmt.Errorf("store clause %s: %w", clausePath, err)
			}
			fmt.Fprintf(w, "  + %s:%s\n", storeName, clausePath)
			imported++
		}
		fmt.Fprintln(w)
	}

	if imported == 0 {
		fmt.Fprintln(w, "No new hooks to import.")
	} else {
		fmt.Fprintf(w, "Imported %d hook(s). Use 'union ratify %s:<prefix>/...' to add them to your contract.\n", imported, storeName)
	}
	return nil
}

func renderHookClause(h harness.Hook, sourceName string) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString("type: hook\n")
	sb.WriteString(fmt.Sprintf("event: %s\n", h.Event))
	if h.Matcher != "" {
		sb.WriteString(fmt.Sprintf("matcher: %s\n", h.Matcher))
	}
	if h.Timeout > 0 {
		sb.WriteString(fmt.Sprintf("timeout: %d\n", h.Timeout))
	}
	sb.WriteString("---\n")
	sb.WriteString(h.Command)
	sb.WriteString("\n")
	return sb.String()
}

func slugify(event, matcher string) string {
	s := strings.ToLower(event)
	s = strings.ReplaceAll(s, " ", "-")
	if matcher != "" {
		s += "-" + strings.ToLower(strings.ReplaceAll(matcher, " ", "-"))
	}
	return s
}
