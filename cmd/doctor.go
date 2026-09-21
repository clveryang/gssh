package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/clveryang/gssh/internal/config"
	"github.com/clveryang/gssh/internal/model"
	"github.com/spf13/cobra"
)

type problem struct {
	Host string
	Msg  string
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the host file for duplicates, typos and missing key files",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := config.Load()
		if err != nil {
			return err
		}
		fmt.Printf("config: %s\n", config.Path())
		problems := lint(c)
		if len(problems) == 0 {
			fmt.Printf("%d hosts, no issues\n", len(c.AllHosts()))
			return nil
		}
		fmt.Println()
		for _, p := range problems {
			fmt.Printf("  %-24s %s\n", p.Host, p.Msg)
		}
		fmt.Printf("\n%d issue(s)\n", len(problems))
		return nil
	},
}

func lint(c *model.Config) []problem {
	var out []problem
	hosts := c.AllHosts()

	seen := map[string]string{}    // lowercase name -> first host that used it
	addrs := map[string][]string{} // address -> host names
	for _, h := range hosts {
		lower := strings.ToLower(h.Name)
		if first, dup := seen[lower]; dup {
			out = append(out, problem{h.Name, fmt.Sprintf("duplicate name (differs from %q only by case)", first)})
		} else {
			seen[lower] = h.Name
		}

		if h.Host == "" {
			out = append(out, problem{h.Name, "no address"})
		}
		if h.Host != "" {
			key := strings.ToLower(h.Host)
			if h.Port != 0 {
				key = fmt.Sprintf("%s:%d", key, h.Port)
			}
			addrs[key] = append(addrs[key], h.Name)
		}

		if h.IdentityFile != "" {
			if p := expand(h.IdentityFile); p != "" {
				if _, err := os.Stat(p); err != nil {
					out = append(out, problem{h.Name, "IdentityFile not found: " + h.IdentityFile})
				}
			}
		}
		if c := shadowedBy(h.Name); c != "" {
			out = append(out, problem{h.Name, shadowHint(h.Name, c)})
		}
		for _, a := range h.Alias {
			if c := shadowedBy(a); c != "" {
				out = append(out, problem{h.Name, "alias " + a + " has the " + shadowHint(a, c)})
			}
		}
		if h.Note == "" && isOpaque(h.Name) {
			out = append(out, problem{h.Name, "opaque name with no note -- you will not remember what this is"})
		}
	}

	for addr, names := range addrs {
		if len(names) > 1 {
			out = append(out, problem{strings.Join(names, ", "), "several names point at " + addr})
		}
	}
	return out
}

// isOpaque reports names that carry no meaning, like "13" or "10.0.4.15".
func isOpaque(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r != '.' && r != ':' && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func expand(p string) string {
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(home, p[1:])
	}
	return p
}

func init() { root.AddCommand(doctorCmd) }
