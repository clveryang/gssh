package cmd

import (
	"fmt"
	"io"

	"github.com/clveryang/gssh/internal/config"
	"github.com/clveryang/gssh/internal/model"
	"github.com/clveryang/gssh/internal/render"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenerate ~/.ssh/config.d/gssh.conf from the YAML host file",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		if len(c.AllHosts()) == 0 {
			return fmt.Errorf("no hosts in %s -- run `gssh import` first", config.Path())
		}
		return runSync(cmd.OutOrStdout(), c, true)
	},
}

// runSync writes the fragment and wires up the Include. It is shared by the
// sync, add and edit commands so that editing a host takes effect immediately.
// When verbose is false it only reports problems, keeping add/edit output short.
func runSync(w io.Writer, c *model.Config, verbose bool) error {
	frag := render.FragmentPath()
	if err := render.WriteFragment(frag, render.Fragment(c, config.Path())); err != nil {
		return err
	}
	if verbose {
		fmt.Fprintf(w, "wrote %d hosts to %s\n", len(c.AllHosts()), frag)
	}

	changed, err := render.EnsureInclude(render.SSHConfigPath(), frag)
	if err != nil {
		return err
	}
	if changed {
		fmt.Fprintf(w, "added Include to %s\n", render.SSHConfigPath())
	}
	if problems := lint(c); len(problems) > 0 {
		fmt.Fprintf(w, "%d issue(s) found -- run `gssh doctor` for detail\n", len(problems))
	}
	return nil
}

func init() { root.AddCommand(syncCmd) }
