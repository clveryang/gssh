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
	importWrite bool
	importFrom  string
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import hosts from an existing ssh_config into the gssh YAML file",
	Long: `Reads an existing ssh_config and converts its Host blocks into the gssh
YAML format.

Nothing is written unless --write is given. Include directives are reported but
not followed: those files belong to other tools. Your ssh_config is never
modified by this command -- run 'gssh sync' when you are happy with the result.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		src := importFrom
		if src == "" {
			src = render.SSHConfigPath()
		}
		res, err := sshconf.ParseFile(src)
		if err != nil {
			return err
		}

		fmt.Printf("read %s\n", src)
		fmt.Printf("  %d host(s)\n", len(res.Hosts))
		for _, w := range res.Wildcards {
			fmt.Printf("  skipped wildcard pattern: Host %s\n", w)
		}
		for _, inc := range res.Includes {
			fmt.Printf("  left alone (owned by another tool): Include %s\n", inc)
		}
		if len(res.Hosts) == 0 {
			return fmt.Errorf("nothing to import")
		}

		dst := config.Path()
		if _, err := os.Stat(dst); err == nil && importWrite {
			return fmt.Errorf("%s already exists -- move it aside first, import will not merge", dst)
		}

		out := &model.Config{Hosts: res.Hosts}
		if !importWrite {
			fmt.Printf("\n--- would write %s ---\n\n", dst)
			data, err := yamlPreview(out)
			if err != nil {
				return err
			}
			fmt.Print(data)
			fmt.Printf("\nnothing written. re-run with --write to save.\n")
			return nil
		}

		// The source file is the only record of these machines; keep a copy.
		if src == render.SSHConfigPath() {
			backup := fmt.Sprintf("%s.bak.%s", src, time.Now().Format("20060102-150405"))
			if data, err := os.ReadFile(src); err == nil {
				if err := os.WriteFile(backup, data, 0o600); err != nil {
					return fmt.Errorf("backup failed, aborting: %w", err)
				}
				fmt.Printf("backed up %s -> %s\n", src, backup)
			}
		}

		if err := config.Save(out); err != nil {
			return err
		}
		fmt.Printf("wrote %d hosts to %s\n", len(res.Hosts), dst)
		fmt.Printf("\nnext: edit it (gssh edit), then `gssh sync`.\n")
		fmt.Printf("your existing ssh_config is untouched until you run sync.\n")
		return nil
	},
}

func init() {
	importCmd.Flags().BoolVar(&importWrite, "write", false, "actually write the YAML file (default: preview only)")
	importCmd.Flags().StringVar(&importFrom, "from", "", "source ssh_config (default: ~/.ssh/config)")
	root.AddCommand(importCmd)
}
