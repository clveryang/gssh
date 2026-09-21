package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/clveryang/gssh/internal/config"
	"github.com/clveryang/gssh/internal/model"
	"github.com/spf13/cobra"
)

var lsTag string

var lsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List hosts (alias: list)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tADDRESS\tGROUP\tTAGS\tNOTE")
		n := 0
		for _, h := range c.AllHosts() {
			if lsTag != "" && !hasTag(h, lsTag) {
				continue
			}
			addr := h.Host
			if h.User != "" {
				addr = h.User + "@" + addr
			}
			if h.Port != 0 {
				addr = fmt.Sprintf("%s:%d", addr, h.Port)
			}
			// One pathological value (a 60-character proxy username, say) would
			// otherwise stretch every column in the table.
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				truncate(h.Name, 24), truncate(addr, 34), h.Group,
				truncate(strings.Join(h.Tags, ","), 16), truncate(h.Note, 32))
			n++
		}
		if err := w.Flush(); err != nil {
			return err
		}
		if n == 0 {
			fmt.Fprintln(os.Stderr, "no hosts matched")
		}
		return nil
	},
}

func hasTag(h *model.Host, tag string) bool {
	for _, t := range h.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func init() {
	lsCmd.Flags().StringVarP(&lsTag, "tag", "t", "", "only show hosts with this tag")
	root.AddCommand(lsCmd)
}
