package term

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const DefaultFontSize = 14

type Settings struct {
	FontSize  float64 `json:"fontSize"`
	Look      string  `json:"look"`
	W         int     `json:"w"`
	H         int     `json:"h"`
	Maximized bool    `json:"maximized"`

	path string
}

func LoadSettings(app string) *Settings {
	s := &Settings{FontSize: DefaultFontSize, Look: "None", W: 1100, H: 760}
	if dir, err := os.UserConfigDir(); err == nil {
		s.path = filepath.Join(dir, app, "settings.json")
		if b, err := os.ReadFile(s.path); err == nil {
			json.Unmarshal(b, s)
		}
	}
	s.FontSize = ClampFontSize(s.FontSize)
	return s
}

func (s *Settings) Save() {
	if s.path == "" {
		return
	}
	os.MkdirAll(filepath.Dir(s.path), 0o755)
	b, _ := json.MarshalIndent(s, "", "  ")
	os.WriteFile(s.path, b, 0o644)
}

func ClampFontSize(sz float64) float64 { return min(40, max(8, sz)) }
