package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/clveryang/gssh/internal/model"
)

// Connecting takes as long as ssh needs to resolve, open a socket and finish
// the handshake, and ssh prints nothing while it does. gssh cannot animate
// through it either: exec replaces this process.
//
// So the connection is made in two steps. First a master connection is opened
// in the background -- it needs no terminal, writes nothing to the screen, and
// exits as soon as it is ready, which is a signal gssh can wait on and animate
// against. Then gssh execs the real session, which attaches to that master and
// is therefore instant. The session itself is still ssh talking to the
// terminal directly: signals, resizes and prompts never pass through gssh.
//
// If anything about the first step does not hold -- no terminal, extra ssh
// arguments, a host that wants a password (BatchMode makes that fail fast
// rather than prompt behind the spinner) -- gssh falls back to exec'ing ssh
// the plain way.

const masterPersist = "60" // seconds the background master lingers, so a second connection is instant

// controlPath is kept short on purpose: a unix socket path is limited to about
// 104 characters, ssh appends a random suffix to it while connecting, and it
// fails with "path too long" when the total does not fit. Under $TMPDIR (long
// on macOS) or a long home directory that limit is easy to hit.
func controlPath(target string) string {
	dir := fmt.Sprintf("/tmp/gssh-%d", os.Getuid())
	sum := sha256.Sum256([]byte(target))
	return filepath.Join(dir, hex.EncodeToString(sum[:])[:16])
}

// socketFits guards against the same limit on systems where even that is too
// long; multiplexing is then skipped rather than failing after a visible wait.
func socketFits(cp string) bool { return len(cp)+18 < 104 }

// masterAlive reports whether a background master for target is already up. A
// socket left behind by a dead master is removed: ssh refuses to open a new
// master over an existing socket file, which would disable multiplexing for
// good.
func masterAlive(target, cp string) bool {
	if _, err := os.Stat(cp); err != nil {
		return false
	}
	if exec.Command("ssh", "-O", "check", "-o", "ControlPath="+cp, target).Run() == nil {
		return true
	}
	os.Remove(cp)
	return false
}

// canAuthWithoutPrompt reports whether ssh is likely to authenticate without
// asking for anything. The master is opened with BatchMode, so a host that
// wants a password fails there and has to be connected again the plain way --
// paying for the handshake twice. Better to skip the master for those.
func canAuthWithoutPrompt(h *model.Host, defaults model.Options) bool {
	if h.IdentityFile != "" || defaults.IdentityFile != "" {
		return true
	}
	return exec.Command("ssh-add", "-l").Run() == nil // agent holds at least one key
}

// openMaster opens the background master, returning false if ssh could not do
// it without a terminal (a password prompt, an unreachable host, and so on).
func openMaster(target, cp string) bool {
	if err := os.MkdirAll(filepath.Dir(cp), 0o700); err != nil {
		return false
	}
	cmd := exec.Command("ssh",
		"-M", "-N", "-f",
		"-o", "ControlPath="+cp,
		"-o", "ControlPersist="+masterPersist,
		"-o", "BatchMode=yes", // fail instead of prompting behind the spinner
		target)
	// No terminal for this one: nothing it might print belongs on screen yet.
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, nil, nil
	return cmd.Run() == nil
}

// dial prepares a connection to h, animating while it does, and returns the
// extra ssh arguments needed to use the result (empty when there is no master).
func dial(h *model.Host, defaults model.Options, target string) []string {
	cp := controlPath(target)
	if !socketFits(cp) {
		return nil
	}
	if masterAlive(target, cp) {
		return []string{"-o", "ControlPath=" + cp} // already connected: no wait to animate
	}
	if !canAuthWithoutPrompt(h, defaults) {
		return nil
	}

	done := make(chan bool, 1)
	go func() { done <- openMaster(target, cp) }()

	ok := spin(os.Stderr, fmt.Sprintf("connecting %s", h.Name), done)
	if !ok {
		return nil
	}
	return []string{"-o", "ControlPath=" + cp}
}

// spin animates until done fires, then erases itself. The animation only ever
// occupies one line, and that line is gone before ssh takes over the terminal.
func spin(w io.Writer, label string, done <-chan bool) bool {
	const fps = 12
	frames := []rune("⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏")
	start := time.Now()
	tick := time.NewTicker(time.Second / fps)
	defer tick.Stop()

	for i := 0; ; i++ {
		select {
		case ok := <-done:
			fmt.Fprint(w, "\r\x1b[2K") // erase the spinner line
			return ok
		case <-tick.C:
			fmt.Fprintf(w, "\r\x1b[2K\x1b[36m%c\x1b[0m %s \x1b[2m%.1fs\x1b[0m",
				frames[i%len(frames)], label, time.Since(start).Seconds())
		}
	}
}
