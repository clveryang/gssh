package cmd

// shadowedBy reports which gssh command a host name collides with, or "".
//
// `gssh ls` always runs the ls command, so a host called "ls" cannot be reached
// that way. It is still reachable -- `gssh -- ls`, the picker, or plain `ssh ls`
// -- but only if someone tells the user, which is what the callers do.
//
// Matching is exact because cobra's command lookup is case-sensitive: a host
// called "LS" is not shadowed.
func shadowedBy(name string) string {
	if name == "help" {
		return "help" // cobra adds the help command lazily, after init
	}
	for _, c := range root.Commands() {
		if c.Name() == name {
			return c.Name()
		}
		for _, a := range c.Aliases {
			if a == name {
				return c.Name()
			}
		}
	}
	return ""
}

func shadowHint(name, cmdName string) string {
	return "same name as the `gssh " + cmdName + "` command -- connect with `gssh -- " +
		name + "` or `ssh " + name + "`, or rename it"
}
