package term

import (
	"fmt"
	"strings"

	caution "github.com/nullentropy/caution"
)

func encodeKey(in caution.TermInput, appCursor bool) string {
	if in.Text != "" {
		return in.Text
	}
	mod := 1
	if in.Shift {
		mod++
	}
	if in.Alt {
		mod += 2
	}
	if in.Ctrl {
		mod += 4
	}
	meta := ""
	if in.Alt {
		meta = "\x1b"
	}
	other := func(code rune) string { return fmt.Sprintf("\x1b[27;%d;%d~", mod, code) }

	if letter, ok := map[string]byte{"Up": 'A', "Down": 'B', "Right": 'C', "Left": 'D', "Home": 'H', "End": 'F'}[in.Key]; ok {
		switch {
		case mod > 1:
			return fmt.Sprintf("\x1b[1;%d%c", mod, letter)
		case appCursor:
			return "\x1bO" + string(letter)
		default:
			return "\x1b[" + string(letter)
		}
	}
	if n, ok := map[string]int{"Insert": 2, "Delete": 3, "PageUp": 5, "PageDown": 6,
		"F5": 15, "F6": 17, "F7": 18, "F8": 19, "F9": 20, "F10": 21, "F11": 23, "F12": 24}[in.Key]; ok {
		if mod > 1 {
			return fmt.Sprintf("\x1b[%d;%d~", n, mod)
		}
		return fmt.Sprintf("\x1b[%d~", n)
	}
	if letter, ok := map[string]byte{"F1": 'P', "F2": 'Q', "F3": 'R', "F4": 'S'}[in.Key]; ok {
		if mod > 1 {
			return fmt.Sprintf("\x1b[1;%d%c", mod, letter)
		}
		return "\x1bO" + string(letter)
	}

	switch in.Key {
	case "Enter":
		if in.Ctrl || in.Shift {
			return other('\r')
		}
		return meta + "\r"
	case "Tab":
		switch {
		case in.Ctrl:
			return other('\t')
		case in.Shift:
			return meta + "\x1b[Z"
		}
		return meta + "\t"
	case "Backspace":
		if in.Ctrl || in.Shift {
			return other(0x7f)
		}
		return meta + "\x7f"
	case "Escape":
		return meta + "\x1b"
	}

	r := []rune(in.Key)
	if len(r) != 1 {
		return ""
	}
	ch := r[0]
	if in.Shift {
		ch = shifted(ch)
	}
	if !in.Ctrl {
		return meta + string(ch)
	}
	switch {
	case ch >= 'a' && ch <= 'z':
		return meta + string(rune(ch-'a'+1))
	case ch == ' ' || ch == '@' || ch == '2':
		return meta + "\x00"
	case ch == '[':
		return meta + "\x1b"
	case ch == '\\':
		return meta + "\x1c"
	case ch == ']':
		return meta + "\x1d"
	case ch == '^' || ch == '6':
		return meta + "\x1e"
	case ch == '/' || ch == '_':
		return meta + "\x1f"
	}
	return other(ch)
}

func shifted(ch rune) rune {
	if ch >= 'a' && ch <= 'z' {
		return ch - 'a' + 'A'
	}
	if i := strings.IndexRune("1234567890-=[]\\;',./`", ch); i >= 0 {
		return []rune("!@#$%^&*()_+{}|:\"<>?~")[i]
	}
	return ch
}
