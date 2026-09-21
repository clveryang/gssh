package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/clveryang/gssh/internal/config"
	"github.com/clveryang/gssh/internal/model"
	"github.com/spf13/cobra"
)

var add struct {
	name, host, user, identity, note, group, jump string
	port                                          int
	tags                                          []string
	alias                                         []string
	noSync                                        bool
}

var addCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a host, interactively or from flags",
	Long: `Adds a host to the YAML file and syncs.

With no flags on a terminal it asks for each field. With --host (and the rest)
it is non-interactive, so it works in scripts. Comments and formatting already
in the host file are preserved.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			add.name = args[0]
		}

		// Interactive only when we have a terminal and the essentials are missing.
		if add.host == "" || add.name == "" {
			if !isTTY() {
				return fmt.Errorf("--host and a name are required when not on a terminal")
			}
			if err := promptForHost(); err != nil {
				return err
			}
		}
		if add.name == "" || add.host == "" {
			return fmt.Errorf("name and host are both required")
		}

		existing, err := config.Load()
		if err != nil {
			return err
		}
		if h, dup := config.Lookup(existing, add.name); dup {
			return fmt.Errorf("host %q already exists (points at %s) -- use `gssh edit` to change it", h.Name, h.Host)
		}

		h := &model.Host{
			Name:  add.name,
			Alias: add.alias,
			Host:  add.host,
			Note:  add.note,
			Tags:  add.tags,
			Options: model.Options{
				User:         add.user,
				Port:         add.port,
				IdentityFile: add.identity,
				ProxyJump:    add.jump,
			},
		}
		if err := config.Insert(h, add.group); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "added %s -> %s\n", h.Name, h.Host)
		if c := shadowedBy(h.Name); c != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "warning: %s\n", shadowHint(h.Name, c))
		}

		if add.noSync {
			fmt.Fprintln(cmd.OutOrStdout(), "not synced (--no-sync); run `gssh sync` to apply")
			return nil
		}
		c, err := config.Load()
		if err != nil {
			return err
		}
		if err := runSync(cmd.OutOrStdout(), c, false); err != nil {
			return err
		}
		if shadowedBy(h.Name) != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "ready: gssh -- %s\n", h.Name)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "ready: gssh %s\n", h.Name)
		}
		return nil
	},
}

func promptForHost() error {
	r := bufio.NewReader(os.Stdin)
	ask := func(label, def string) string {
		if def != "" {
			fmt.Fprintf(os.Stderr, "%s [%s]: ", label, def)
		} else {
			fmt.Fprintf(os.Stderr, "%s: ", label)
		}
		line, err := r.ReadString('\n')
		if err != nil {
			return def
		}
		if v := strings.TrimSpace(line); v != "" {
			return v
		}
		return def
	}

	add.name = ask("name (what you will type after `gssh`)", add.name)
	add.host = ask("address (IP or hostname)", add.host)
	add.user = ask("user", add.user)
	if p := ask("port", portOrEmpty(add.port)); p != "" {
		n, err := strconv.Atoi(p)
		if err != nil {
			return fmt.Errorf("port %q is not a number", p)
		}
		add.port = n
	}
	add.identity = ask("identity file (blank to use defaults)", add.identity)
	add.note = ask("note (shown in the picker and in completion)", add.note)
	if t := ask("tags (comma separated)", strings.Join(add.tags, ",")); t != "" {
		add.tags = splitComma(t)
	}
	add.group = ask("group (blank for none)", add.group)
	return nil
}

func portOrEmpty(p int) string {
	if p == 0 {
		return ""
	}
	return strconv.Itoa(p)
}

func splitComma(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func init() {
	f := addCmd.Flags()
	f.StringVar(&add.host, "host", "", "address (IP or hostname)")
	f.StringVar(&add.user, "user", "", "login user")
	f.IntVar(&add.port, "port", 0, "port")
	f.StringVar(&add.identity, "identity-file", "", "private key path")
	f.StringVar(&add.jump, "jump", "", "ProxyJump host")
	f.StringVar(&add.note, "note", "", "what this machine is for")
	f.StringSliceVar(&add.tags, "tag", nil, "tag (repeatable)")
	f.StringSliceVar(&add.alias, "alias", nil, "alternative name (repeatable)")
	f.StringVar(&add.group, "group", "", "group to add it to (created if missing)")
	f.BoolVar(&add.noSync, "no-sync", false, "do not sync after adding")
	root.AddCommand(addCmd)
}
