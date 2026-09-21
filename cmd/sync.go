package cmd

import (
	"fmt"

	"github.com/clveryang/gssh/internal/config"
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
		hosts := c.AllHosts()
		if len(hosts) == 0 {
			return fmt.Errorf("no hosts in %s -- run `gssh import` first", config.Path())
		}

		frag := render.FragmentPath()
		if err := render.WriteFragment(frag, render.Fragment(c, config.Path())); err != nil {
			return err
		}
		fmt.Printf("wrote %d hosts to %s\n", len(hosts), frag)

		changed, err := render.EnsureInclude(render.SSHConfigPath(), frag)
		if err != nil {
			return err
		}
		if changed {
			fmt.Printf("added Include to %s\n", render.SSHConfigPath())
		}
		if problems := lint(c); len(problems) > 0 {
			fmt.Printf("\n%d issue(s) found -- run `gssh doctor` for detail\n", len(problems))
		}
		return nil
	},
}

func init() { root.AddCommand(syncCmd) }
