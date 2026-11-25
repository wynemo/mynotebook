package notebook

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Preferences is kept separately from the workspace so either can be reset.
type Preferences struct {
	Theme            string  `json:"theme"`
	FontSize         float32 `json:"fontSize"`
	Wrap             bool    `json:"wrap"`
	DefaultMode      int     `json:"defaultMode"`
	AutomaticUpdates bool    `json:"automaticUpdates"`
}

func DefaultPreferences() Preferences {
	return Preferences{Theme: "system", FontSize: 14, Wrap: true, AutomaticUpdates: true}
}
func (p *Preferences) normalize() {
	if p.Theme != "system" && p.Theme != "light" && p.Theme != "dark" {
		p.Theme = "system"
	}
	if p.FontSize < 10 || p.FontSize > 24 {
		p.FontSize = 14
	}
	if p.DefaultMode < 0 || p.DefaultMode > 2 {
		p.DefaultMode = 0
	}
}
func LoadPreferences(path string) (Preferences, error) {
	p := DefaultPreferences()
	b, e := os.ReadFile(path)
	if os.IsNotExist(e) {
		return p, nil
	}
	if e != nil {
		return p, e
	}
	e = json.Unmarshal(b, &p)
	p.normalize()
	return p, e
}
func SavePreferences(path string, p Preferences) error {
	p.normalize()
	b, e := json.MarshalIndent(p, "", "  ")
	if e != nil {
		return e
	}
	if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	return AtomicWrite(path, b, 0600)
}
