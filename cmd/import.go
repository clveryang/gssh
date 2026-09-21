package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/clveryang/gssh/internal/config"
	"github.com/clveryang/gssh/internal/model"
	"github.com/clveryang/gssh/internal/render"
	"github.com/clveryang/gssh/internal/sshconf"
	"github.com/spf13/cobra"
)

var (
	importYes         bool
	importDryRun      bool
	importFrom        string
	importWriteCompat bool
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import hosts from your existing ~/.ssh/config",
	Long: `Converts the Host blocks in an existing ssh_config into the gssh YAML file,
then syncs.

It shows what it found and asks before writing. Include directives are reported
but not followed: those files belong to other tools. Your ssh_config is backed
up before anything is written, and keeps all of its original content -- gssh
only adds one Include line.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		src := importFrom
		if src == "" {
			src = render.SSHConfigPath()
		}
		res, err := sshconf.ParseFile(src)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "read %s\n", src)
		fmt.Fprintf(out, "  %d host(s)\n", len(res.Hosts))
		for _, w := range res.Wildcards {
			fmt.Fprintf(out, "  skipped wildcard: Host %s\n", w)
		}
		for _, inc := range res.Includes {
			fmt.Fprintf(out, "  left alone (another tool owns it): Include %s\n", inc)
		}
		if len(res.Hosts) == 0 {
			return fmt.Errorf("nothing to import")
		}

		cfg := &model.Config{Hosts: res.Hosts}
		dst := config.Path()

		if importDryRun {
			fmt.Fprintf(out, "\n--- %s would contain ---\n\n", dst)
			data, err := yamlPreview(cfg)
			if err != nil {
				return err
			}
			fmt.Fprint(out, data)
			return nil
		}

		fmt.Fprintln(out)
		printHostSummary(out, res.Hosts)

		if _, err := os.Stat(dst); err == nil {
			return fmt.Errorf("%s already exists -- import will not merge; move it aside or use `gssh edit`", dst)
		}

		if !importYes {
			if !isTTY() {
				return fmt.Errorf("refusing to write without confirmation; pass --yes")
			}
			if !confirm(fmt.Sprintf("\nimport %d hosts into %s?", len(res.Hosts), dst)) {
				fmt.Fprintln(out, "cancelled; nothing written")
				return nil
			}
		}

		// The source is the only record of how to reach these machines.
		if src == render.SSHConfigPath() {
			backup := fmt.Sprintf("%s.bak.%s", src, time.Now().Format("20060102-150405"))
			if data, err := os.ReadFile(src); err == nil {
				if err := os.WriteFile(backup, data, 0o600); err != nil {
					return fmt.Errorf("backup failed, aborting: %w", err)
				}
				fmt.Fprintf(out, "backed up %s -> %s\n", src, backup)
			}
		}

		if err := config.Save(cfg); err != nil {
			return err
		}
		fmt.Fprintf(out, "wrote %d hosts to %s\n", len(res.Hosts), dst)

		loaded, err := config.Load()
		if err != nil {
			return err
		}
		// verbose: runSync reports the fragment it wrote and any lint issues,
		// and those belong at the end rather than before the sync line.
		if err := runSync(out, loaded, true); err != nil {
			return err
		}
		fmt.Fprintf(out, "\ndone. try `gssh` to pick a host, or `gssh write` to edit them.\n")
		return nil
	},
}

// printHostSummary lists what is about to be imported, briefly. Dumping 29
// hosts of YAML at someone before a prompt is not a preview, it is noise.
func printHostSummary(out interface{ Write([]byte) (int, error) }, hosts []*model.Host) {
	const show = 8
	for i, h := range hosts {
		if i == show {
			fmt.Fprintf(out, "  ... and %d more\n", len(hosts)-show)
			break
		}
		addr := h.Host
		if h.User != "" {
			addr = h.User + "@" + addr
		}
		if h.Port != 0 {
			addr = fmt.Sprintf("%s:%d", addr, h.Port)
		}
		fmt.Fprintf(out, "  %-22s %s\n", truncate(h.Name, 22), truncate(addr, 44))
	}
	fmt.Fprintf(out, "\n(full YAML: gssh import --dry-run)\n")
}

func truncate(s string, n int) string {
	if len([]rune(s)) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

func init() {
	f := importCmd.Flags()
	f.BoolVarP(&importYes, "yes", "y", false, "do not ask for confirmation")
	f.BoolVar(&importDryRun, "dry-run", false, "print the resulting YAML and exit")
	f.StringVar(&importFrom, "from", "", "source ssh_config (default: ~/.ssh/config)")
	// --write used to be required. Keep it working so anyone following the old
	// instructions is not met with an error.
	f.BoolVar(&importWriteCompat, "write", false, "")
	f.MarkDeprecated("write", "writing is now the default; use --dry-run to preview")
	root.AddCommand(importCmd)
}
