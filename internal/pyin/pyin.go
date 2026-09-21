// Package pyin derives searchable latin keys for names containing Han characters,
// so that "杭州备份机" can be reached by typing "hangzhoubeifenji" or "hzbfj".
package pyin

import (
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
)

var arg = pinyin.NewArgs() // default: no tone marks, first reading only

// Keys returns the extra lowercase search keys for s: the full pinyin and the
// initials. Returns nil when s has no Han characters, so ASCII names cost nothing.
func Keys(s string) []string {
	if !hasHan(s) {
		return nil
	}
	var full, initials strings.Builder
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			py := pinyin.SinglePinyin(r, arg)
			if len(py) == 0 || py[0] == "" {
				continue
			}
			full.WriteString(py[0])
			initials.WriteByte(py[0][0])
			continue
		}
		// Keep ASCII/digits inline: "广州生产1" -> "guangzhoushengchan1".
		if r < unicode.MaxASCII {
			lower := unicode.ToLower(r)
			full.WriteRune(lower)
			initials.WriteRune(lower)
		}
	}
	keys := make([]string, 0, 2)
	if f := full.String(); f != "" {
		keys = append(keys, f)
	}
	if i := initials.String(); i != "" && i != full.String() {
		keys = append(keys, i)
	}
	return keys
}

func hasHan(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
