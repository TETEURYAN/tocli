package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
)

// namedKeys are the bytes a typical terminal sends for keys that are not plain characters.
var namedKeys = map[string]string{
	"esc":       "\x1b",
	"tab":       "\t",
	"shift+tab": "\x1b[Z",
	"enter":     "\r",
	"space":     " ",
	"backspace": "\x7f",
	"delete":    "\x1b[3~",
	"up":        "\x1b[A",
	"down":      "\x1b[B",
	"right":     "\x1b[C",
	"left":      "\x1b[D",
	"home":      "\x1b[H",
	"end":       "\x1b[F",
	"pgup":      "\x1b[5~",
	"pgdown":    "\x1b[6~",
}

// termBytes returns the bytes a terminal sends for the named key: a name from namedKeys,
// "ctrl+<letter>", or a single character.
func termBytes(name string) ([]byte, error) {
	if b, ok := namedKeys[name]; ok {
		return []byte(b), nil
	}
	if letter, ok := strings.CutPrefix(name, "ctrl+"); ok && len(letter) == 1 && letter[0] >= 'a' && letter[0] <= 'z' {
		return []byte{letter[0] - 'a' + 1}, nil
	}
	if r := []rune(name); len(r) == 1 {
		return []byte(name), nil
	}
	return nil, fmt.Errorf("unknown key name %q", name)
}

// termKey is the message Bubble Tea delivers to Update when the user presses the named key. It is
// produced by the same decoder the real program uses, from the bytes a terminal sends.
func termKey(t testing.TB, name string) tea.KeyPressMsg {
	t.Helper()
	b, err := termBytes(name)
	if err != nil {
		t.Fatal(err)
	}
	var d uv.EventDecoder
	n, ev := d.Decode(b)
	press, ok := ev.(uv.KeyPressEvent)
	if !ok || n != len(b) {
		t.Fatalf("%q (% x) decoded to %T consuming %d of %d bytes", name, b, ev, n, len(b))
	}
	return tea.KeyPressMsg(press)
}
