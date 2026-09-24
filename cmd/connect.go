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
	"github.com/clveryang/gssh/internal/pyin"
	"github.com/clveryang/gssh/internal/render"
	"github.com/clveryang/gssh/internal/sshconf"
	"github.com/clveryang/gssh/internal/ui"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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

	return connect(h, c.Defaults, rest)
}

// connect execs ssh, replacing this process. It never returns on success.
func connect(h *model.Host, defaults model.Options, extra []string) error {
	bin, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("ssh not found in PATH: %w", err)
	}
	// Recorded before the exec, because after it there is no "after".
	mru.Touch(h.Name)

	// h.Target, not h.Name: ssh rejects non-ASCII names outright.
	target := h.Target()

	// Extra ssh arguments (port forwards and the like) belong to the session,
	// not to a shared master, so they take the plain path.
	var via []string
	if len(extra) == 0 && useMultiplexing() {
		via = dial(h, defaults, target)
	}
	if via == nil {
		announce(h)
	}

	args := append([]string{"ssh"}, via...)
	args = append(args, target)
	args = append(args, extra...)
	return syscall.Exec(bin, args, os.Environ())
}

// announce prints what is being connected to before handing over to ssh.
//
// There is no spinner to show here: exec replaces this process, so after the
// handover gssh no longer exists, and the pause the user sees is ssh resolving
// DNS, opening the connection and doing the handshake -- during which ssh
// prints nothing. A line printed first means the screen is never just blank.
func useMultiplexing() bool {
	return os.Getenv("GSSH_NO_MULTIPLEX") == "" && term.IsTerminal(int(os.Stderr.Fd()))
}

func announce(h *model.Host) {
	if !term.IsTerminal(int(os.Stderr.Fd())) {
		return
	}
	addr := h.Host
	if h.User != "" {
		addr = h.User + "@" + addr
	}
	if h.Port != 0 {
		addr = fmt.Sprintf("%s:%d", addr, h.Port)
	}
	label := h.Name
	if h.Note != "" {
		label += "  " + h.Note
	}
	// Dim, so it reads as gssh's own line rather than ssh output.
	fmt.Fprintf(os.Stderr, "\x1b[2m→\x1b[0m %s \x1b[2m%s\x1b[0m\n", label, addr)
}

// completeHosts powers shell completion of host names.
//
// Matches come in tiers and only the best non-empty tier is returned, because
// the zsh script hands the result to compadd -U and does no filtering of its
// own. Without tiers, `gssh we<TAB>` would offer every host containing a "v"
// instead of completing to the one host whose name starts with it.
//
//  1. name or alias starts with the input, case-sensitively
//  2. the same, ignoring case                  v     -> web-01
//  3. pinyin of the name starts with it        hzbfj  -> 杭州备份机
//  4. input appears anywhere (IP, note, tag)   0.0.0 -> 42, 14, ...
//
// Tier 1 exists because compadd -U replaces the typed word with the common
// prefix of the candidates: "Tenc" matching both Tencent and tencent would
// share no prefix at all and erase what the user typed.
func completeHosts(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	c, err := config.Load()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var tiers [4][]string
	for _, h := range c.AllHosts() {
		t := matchTier(h, toComplete)
		if t < 0 {
			continue
		}
		// "name\tdescription": zsh shows the description beside the name,
		// which is the point -- "13" alone tells you nothing.
		desc := h.Note
		if desc == "" {
			desc = h.Host
		}
		tiers[t] = append(tiers[t], h.Name+"\t"+desc)
	}
	for _, t := range tiers {
		if len(t) > 0 {
			return t, cobra.ShellCompDirectiveNoFileComp
		}
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// matchTier returns the tier (0-3) described on completeHosts, or -1.
func matchTier(h *model.Host, typed string) int {
	if typed == "" {
		return 0
	}
	names := append([]string{h.Name}, h.Alias...)
	for _, n := range names {
		if strings.HasPrefix(n, typed) {
			return 0
		}
	}
	want := strings.ToLower(typed)
	for _, n := range names {
		if strings.HasPrefix(strings.ToLower(n), want) {
			return 1
		}
	}
	keys := pyin.Keys(h.Name)
	if h.Pinyin != "" {
		keys = append(keys, strings.ToLower(h.Pinyin))
	}
	for _, k := range keys {
		if strings.HasPrefix(k, want) {
			return 2
		}
	}
	for _, s := range h.Search {
		if strings.Contains(s, want) {
			return 3
		}
	}
	return -1
}

// firstRun handles `gssh` with nothing configured yet. Rather than printing an
// error and a list of commands to go read about, it offers to do the obvious
// thing: import the hosts the user already has.
func firstRun(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	sshCfg := render.SSHConfigPath()

	if res, err := sshconf.ParseFile(sshCfg); err == nil && len(res.Hosts) > 0 && isTTY() {
		fmt.Fprintf(out, "No hosts configured yet, but %s already has %d.\n", sshCfg, len(res.Hosts))
		if confirm("Import them now?") {
			importYes = true // they just said yes; do not ask twice
			return importCmd.RunE(cmd, nil)
		}
		fmt.Fprintf(out, "\nok. `gssh import` when you want to, or `gssh write` to add them by hand.\n")
		return nil
	}

	return fmt.Errorf("no hosts yet in %s\n  `gssh write`  to add some by hand\n  `gssh import` to take them from ~/.ssh/config", config.Path())
}

// runPicker is the no-argument entry point: show the interactive list.
func runPicker(cmd *cobra.Command) error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	hosts := c.AllHosts()
	if len(hosts) == 0 {
		return firstRun(cmd)
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
		return connect(res.Host, c.Defaults, nil)
	case ui.ActionEdit:
		return editCmd.RunE(cmd, nil)
	}
	return nil // quit without choosing
}
