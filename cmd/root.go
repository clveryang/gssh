// Package cmd implements the gssh command line.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is overridden at build time via -ldflags.
var Version = "dev"

var root = &cobra.Command{
	Use:   "gssh [host] [-- ssh-args...]",
	Short: "Visual ssh host manager on top of ~/.ssh/config",
	Long: `gssh keeps your hosts in one YAML file, renders them into an ssh_config
fragment, and gives you a searchable picker over them.

With a host name it connects directly. With no arguments it opens the picker.`,
	Version:       Version,
	Args:          cobra.ArbitraryArgs,
	SilenceUsage:  true,
	SilenceErrors: true, // Execute prints the error itself
	// Let cobra complete host names for the bare `gssh <TAB>` form.
	ValidArgsFunction: completeHosts,
}

func init() {
	// Assigned here rather than in the literal: runConnect reaches back to
	// root (via shadowedBy), and Go rejects that as an initialisation cycle.
	root.RunE = runConnect
}

// Execute runs the CLI.
func Execute() {
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "gssh:", err)
		os.Exit(1)
	}
}
