package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// A unix socket path is capped around 104 bytes and ssh appends a suffix to it
// while connecting; when it does not fit, ssh fails after the user has already
// watched the spinner, and the connection is then made twice.
func TestControlPathFitsInASocket(t *testing.T) {
	cp := controlPath("some.long.host.name.example.com")
	if !socketFits(cp) {
		t.Errorf("control path too long for a socket: %d bytes, %q", len(cp), cp)
	}
	if got := controlPath("a"); got == controlPath("b") {
		t.Error("different hosts must not share a control path")
	}
	if controlPath("a") != controlPath("a") {
		t.Error("control path must be stable across runs")
	}
}

func TestSocketFits(t *testing.T) {
	if socketFits("/tmp/" + strings.Repeat("x", 100)) {
		t.Error("an over-long path should be rejected")
	}
}

// The spinner must leave the line clean: ssh draws on that terminal next.
func TestSpinErasesItself(t *testing.T) {
	var buf bytes.Buffer
	done := make(chan bool, 1)
	go func() { time.Sleep(250 * time.Millisecond); done <- true }()

	if ok := spin(&buf, "connecting x", done); !ok {
		t.Error("spin should return what done carried")
	}
	out := buf.String()
	if !strings.Contains(out, "connecting x") {
		t.Errorf("no label drawn: %q", out)
	}
	if !strings.HasSuffix(out, "\r\x1b[2K") {
		t.Errorf("spinner did not erase its line, output ends with %q", out[max(0, len(out)-12):])
	}
}

func TestSpinReturnsFailure(t *testing.T) {
	done := make(chan bool, 1)
	done <- false
	if spin(&bytes.Buffer{}, "x", done) {
		t.Error("spin should report failure so the caller can fall back")
	}
}
