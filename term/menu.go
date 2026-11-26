package term

import (
	"log"
	"os/exec"

	caution "github.com/nullentropy/caution"
)

func ViewMenu(t *Window, changed func()) caution.Menu {
	done := func() {
		if changed != nil {
			changed()
		}
	}
	var looks []caution.MenuItem
	current := LookIndex(t.Look())
	for i, l := range Looks {
		title := "    " + l.Name
		if i == current {
			title = "✓  " + l.Name
		}
		looks = append(looks, caution.MenuItem{Title: title, OnPick: func() { t.SetLook(l.Name); done() }})
	}
	step := func(d int) func() {
		return func() {
			n := len(Looks)
			t.SetLook(Looks[((LookIndex(t.Look())+d)%n+n)%n].Name)
			done()
		}
	}
	return caution.Menu{Title: "View", Items: []caution.MenuItem{
		{Title: "Bigger", Key: "cmd+=", OnPick: func() { t.SetFontSize(t.FontSize() + 1); done() }},
		{Title: "Smaller", Key: "cmd+-", OnPick: func() { t.SetFontSize(t.FontSize() - 1); done() }},
		{Title: "Actual Size", Key: "cmd+0", OnPick: func() { t.SetFontSize(DefaultFontSize); done() }},
		{Sep: true},
		{Title: "Look", Items: looks},
		{Title: "Next Look", Key: "cmd+]", OnPick: step(1)},
		{Title: "Previous Look", Key: "cmd+[", OnPick: step(-1)},
	}}
}

func (t *Term) PasteClipboard() {
	go func() {
		out, err := exec.Command("pbpaste").Output()
		if err != nil {
			log.Printf("paste: %v", err)
			return
		}
		t.Paste(string(out))
	}()
}
