package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/clveryang/gssh/internal/config"
	"github.com/clveryang/gssh/internal/model"
	"github.com/clveryang/gssh/internal/mru"
	"github.com/clveryang/gssh/internal/ui"
	"github.com/spf13/cobra"
)

// runConnect handles `gssh <host> [extra ssh args]`.
//
// It never speaks SSH itself: it execs the real ssh binary so that agent
// forwarding, ControlMaster, tty handling and signals behave exactly as if the
// user had typed ssh directly.
func runConnect(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runPicker(cmd)
	}

	c, err := config.Load()
	if err != nil {
		return err
	}
	name := args[0]
	h, ok := config.Lookup(c, name)
	if !ok {
		return fmt.Errorf("unknown host %q -- `gssh ls` to see them all", name)
	}

	// Everything after the host (and after a literal --) goes to ssh verbatim.
	rest := args[1:]
	if d := cmd.ArgsLenAtDash(); d > 0 {
		rest = args[d:]
	}

	return connect(h, rest)
}

// connect execs ssh, replacing this process. It never returns on success.
func connect(h *model.Host, extra []string) error {
	bin, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}
	// Recorded before the exec, because after it there is no "after".
	mru.Touch(h.Name)
	return syscall.Exec(bin, append([]string{"ssh", h.Name}, extra...), os.Environ())
}

// completeHosts powers shell completion of host names. It also matches on
// pinyin, so typing `hzbfj<TAB>` can reach 杭州备份机.
func completeHosts(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	c, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	want := strings.ToLower(toComplete)
	var out []string
	for _, h := range c.AllHosts() {
		if want != "" && !matches(h.Search, want) {
			continue
		}
		// "name\tdescription" -- zsh and fish show the description inline,
		// which is the whole point: the name alone is unreadable.
		desc := h.Note
		if desc == "" {
			desc = h.Host
		}
		out = append(out, h.Name+"\t"+desc)
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func matches(haystacks []string, want string) bool {
	for _, s := range haystacks {
		if strings.Contains(s, want) {
			return true
		}
	}
	return false
}

// runPicker is the no-argument entry point: show the interactive list.
func runPicker(cmd *cobra.Command) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	hosts := c.AllHosts()
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts in %s -- run `gssh import` or `gssh add`", config.Path())
	}
	// Without a terminal there is nothing to drive the picker with; a plain
	// list keeps `gssh | grep ...` working.
	if !isTTY() {
		return lsCmd.RunE(cmd, nil)
	}

	res, err := ui.Run(hosts)
	if err != nil {
		return err
	}
	switch res.Action {
	case ui.ActionConnect:
		return connect(res.Host, nil)
	case ui.ActionEdit:
		return editCmd.RunE(cmd, nil)
	}
	return nil // quit without choosing
}
