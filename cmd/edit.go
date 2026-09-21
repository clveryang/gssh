package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/clveryang/gssh/internal/config"
	"github.com/clveryang/gssh/internal/model"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"
)

var editNoSync bool

var editCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the YAML host file in $EDITOR, then validate and sync",
	Long: `Opens the host file in $VISUAL, $EDITOR, or vi.

The file is edited through a temporary copy: it is only written back once it
parses, so a syntax error can never leave you with a broken host file. On a
successful edit gssh syncs automatically unless --no-sync is given.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.Path()
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := writeTemplate(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", path)
		}

		original, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		tmp, err := os.CreateTemp("", "gssh-*.yaml")
		if err != nil {
			return err
		}
		defer os.Remove(tmp.Name())
		if _, err := tmp.Write(original); err != nil {
			tmp.Close()
			return err
		}
		tmp.Close()

		for {
			if err := launchEditor(tmp.Name()); err != nil {
				return err
			}
			edited, err := os.ReadFile(tmp.Name())
			if err != nil {
				return err
			}

			var parsed model.Config
			if err := yaml.Unmarshal(edited, &parsed); err != nil {
				fmt.Fprintf(os.Stderr, "\n%s is not valid YAML:\n  %v\n\n", filepath.Base(path), err)
				if !isTTY() || !confirm("edit again?") {
					return fmt.Errorf("not saved; your original %s is untouched", path)
				}
				continue
			}

			if string(edited) == string(original) {
				fmt.Fprintln(cmd.OutOrStdout(), "no changes")
				return nil
			}
			if err := os.WriteFile(path, edited, 0o600); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "saved %s\n", path)

			if editNoSync {
				fmt.Fprintln(cmd.OutOrStdout(), "not synced (--no-sync); run `gssh sync` to apply")
				return nil
			}
			// Reload through config.Load so groups and tags are indexed the
			// same way every other command sees them.
			c, err := config.Load()
			if err != nil {
				return err
			}
			return runSync(cmd.OutOrStdout(), c, false)
		}
	},
}

func launchEditor(path string) error {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	// Respect things like EDITOR="code --wait".
	parts := strings.Fields(editor)
	c := exec.Command(parts[0], append(parts[1:], path)...)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("editor %q failed: %w", editor, err)
	}
	return nil
}

func isTTY() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s [Y/n] ", prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "", "y", "yes":
		return true
	}
	return false
}

const templateYAML = `# gssh hosts -- edit this file, then run: gssh sync
#
# defaults fill in any field a host leaves unset.
defaults:
  identity_file: ~/.ssh/id_rsa
  server_alive_interval: 30

groups:
  - name: example
    tags: [demo]
    hosts:
      - name: shanghai
        host: 10.0.1.13
        user: deploy
        note: what this machine is for
`

func writeTemplate(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(templateYAML), 0o600)
}

func init() {
	editCmd.Flags().BoolVar(&editNoSync, "no-sync", false, "do not run sync after a successful edit")
	root.AddCommand(editCmd)
}
