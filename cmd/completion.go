package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// zshHostCompletion replaces cobra's handling of the first argument.
//
// Cobra's script feeds candidates to _describe, and zsh then keeps only those
// that start with what was typed. That silently drops the matches gssh exists
// for -- pinyin (hzbfj -> 杭州备份机) and IP fragments (0.0.0 -> 42) -- because
// the host name does not start with the input. compadd -U hands the decision to
// gssh, which already returns only the best tier of matches.
//
// Everything after the first argument (subcommand flags and so on) still goes
// through cobra's generated function.
const zshHostCompletion = `
_gssh() {
  if (( CURRENT != 2 )); then
    _gssh_cobra "$@"
    return
  fi
  local -a names descs
  local line name desc
  for line in "${(@f)$(${~words[1]} __complete "${words[CURRENT]}" 2>/dev/null)}"; do
    [[ -z $line || $line == :* ]] && continue
    name=${line%%$'\t'*}
    desc=${line#*$'\t'}
    [[ $desc == "$line" ]] && desc=""
    names+=("$name")
    descs+=("${name}${desc:+  -- $desc}")
  done
  (( $#names )) || return 1
  compadd -U -l -d descs -a names
}
compdef _gssh gssh
`

// bashCompletion is self-contained rather than cobra's generated script, which
// needs the bash-completion package (_get_comp_words_by_ref). macOS and minimal
// servers do not have it, and the script then fails silently. Bash uses a
// completion function's COMPREPLY as given, so pinyin matches survive here too.
// Written for bash 3.2, which is what macOS still ships.
const bashCompletion = `_gssh_complete() {
  local line out
  COMPREPLY=()
  out=$("${COMP_WORDS[0]}" __complete "${COMP_WORDS[@]:1:COMP_CWORD}" 2>/dev/null) || return 0
  # IFS is changed only after the call: in bash 3.2 a non-default IFS makes the
  # quoted array slice above collapse into a single argument.
  local IFS=$'\n'
  for line in $out; do
    [[ -z $line || $line == :* ]] && continue
    COMPREPLY+=("${line%%$'\t'*}")
  done
}
complete -F _gssh_complete gssh
`

var completionCmd = &cobra.Command{
	Use:   "completion [zsh|bash|fish]",
	Short: "Print the shell completion script",
	Long: `Prints a completion script. For zsh, add this line to ~/.zshrc after compinit:

  source <(gssh completion zsh)

Then "gssh v<TAB>" completes host names, and pinyin works too: "gssh hzbfj<TAB>".`,
	ValidArgs: []string{"zsh", "bash", "fish"},
	Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		shell := "zsh"
		if len(args) == 1 {
			shell = args[0]
		}
		out := cmd.OutOrStdout()
		switch shell {
		case "bash":
			_, err := fmt.Fprint(out, bashCompletion)
			return err
		case "fish":
			return root.GenFishCompletion(out, true)
		}

		var buf bytes.Buffer
		if err := root.GenZshCompletion(&buf); err != nil {
			return err
		}
		script := buf.String()
		// Keep cobra's function under another name for everything past the
		// first argument, and register ours instead of it.
		renamed := strings.Replace(script, "\n_gssh()", "\n_gssh_cobra()", 1)
		renamed = strings.Replace(renamed, "compdef _gssh gssh\n", "", 1)
		if renamed == script || !strings.Contains(renamed, "_gssh_cobra()") {
			return fmt.Errorf("unexpected cobra zsh script layout; cannot install host completion")
		}
		_, err := fmt.Fprint(out, renamed+zshHostCompletion)
		return err
	},
}

func init() {
	root.CompletionOptions.DisableDefaultCmd = true
	root.AddCommand(completionCmd)
}
