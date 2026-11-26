package term

import (
	"encoding/base64"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nullentropy/caution/native"
)

func PrepareBundle() string {
	if !native.InBundle() {
		return ""
	}
	adoptLoginPath()
	home, _ := os.UserHomeDir()
	return home
}

func adoptLoginPath() {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/zsh"
	}
	out, err := exec.Command(shell, "-ilc", `printf '\n__PATH__%s__PATH__' "$PATH"`).Output()
	if err != nil {
		return
	}
	s := string(out)
	i := strings.Index(s, "__PATH__")
	j := strings.LastIndex(s, "__PATH__")
	if i >= 0 && j > i {
		os.Setenv("PATH", s[i+len("__PATH__"):j])
	}
}

func LoadFonts(glob string) []string {
	if glob == "" {
		return nil
	}
	if strings.HasPrefix(glob, "~/") {
		home, _ := os.UserHomeDir()
		glob = filepath.Join(home, glob[2:])
	}
	var paths []string
	if i, j := strings.Index(glob, "{"), strings.Index(glob, "}"); i >= 0 && j > i {
		for _, alt := range strings.Split(glob[i+1:j], ",") {
			paths = append(paths, glob[:i]+alt+glob[j+1:])
		}
	} else {
		paths = []string{glob}
	}
	var srcs []string
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			log.Printf("font %s: %v", p, err)
			continue
		}
		srcs = append(srcs, "data:font/ttf;base64,"+base64.StdEncoding.EncodeToString(b))
	}
	return srcs
}

const DefaultFonts = "~/Library/Fonts/FiraCodeNerdFontMono-{Regular,Bold}.ttf"
