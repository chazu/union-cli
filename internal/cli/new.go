package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/chazu/union/internal/paths"
	"github.com/chazu/union/internal/store"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newNewCmd() *cobra.Command {
	var fromFile string
	cmd := &cobra.Command{
		Use:   "new <path>",
		Short: "Author a new clause (editor by default; stdin if piped; -f to seed from a file).",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openStore()
			if err != nil {
				return err
			}
			path := args[0]
			if s.Has(path) {
				return fmt.Errorf("clause already exists: %s (use 'union edit' to change it)", path)
			}
			body, err := readClauseInput(fromFile, path)
			if err != nil {
				return err
			}
			if err := s.Put(path, body, "new "+path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created clause %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVarP(&fromFile, "file", "f", "", "seed clause body from FILE (use '-' for stdin)")
	return cmd
}

// readClauseInput resolves -f, stdin, or $EDITOR.
func readClauseInput(fromFile, clausePath string) ([]byte, error) {
	if fromFile == "-" {
		return io.ReadAll(os.Stdin)
	}
	if fromFile != "" {
		return os.ReadFile(fromFile)
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return io.ReadAll(os.Stdin)
	}
	return openEditor(nil, clausePath)
}

func openEditor(seed []byte, clausePath string) ([]byte, error) {
	ed := os.Getenv("VISUAL")
	if ed == "" {
		ed = os.Getenv("EDITOR")
	}
	if ed == "" {
		ed = "vi"
	}
	tmp, err := os.CreateTemp("", "union-*-"+filepath.Base(clausePath)+".md")
	if err != nil {
		return nil, fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if len(seed) > 0 {
		if _, err := tmp.Write(seed); err != nil {
			tmp.Close()
			return nil, err
		}
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	c := exec.Command(ed, tmpPath)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("editor exited with error: %w", err)
	}
	return os.ReadFile(tmpPath)
}

func openStore() (*store.Store, error) {
	dir, err := paths.UnionDir()
	if err != nil {
		return nil, err
	}
	return store.Open(dir)
}
