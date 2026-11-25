package notebook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPreferencesDefaultsMigrationAndRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings.json")
	got, e := LoadPreferences(p)
	if e != nil || !got.Wrap || got.FontSize != 14 || !got.AutomaticUpdates {
		t.Fatalf("defaults: %+v %v", got, e)
	}
	if e = os.WriteFile(p, []byte(`{"theme":"dark"}`), 0600); e != nil {
		t.Fatal(e)
	}
	got, e = LoadPreferences(p)
	if e != nil || got.Theme != "dark" || !got.Wrap {
		t.Fatalf("partial: %+v %v", got, e)
	}
	got.FontSize = 18
	got.Wrap = false
	got.DefaultMode = 2
	got.AutomaticUpdates = false
	if e = SavePreferences(p, got); e != nil {
		t.Fatal(e)
	}
	restored, e := LoadPreferences(p)
	if e != nil || restored != got {
		t.Fatalf("round trip: %+v %v", restored, e)
	}
	if e = os.WriteFile(p, []byte(`{"theme":"invalid","fontSize":999,"defaultMode":-1}`), 0600); e != nil {
		t.Fatal(e)
	}
	got, e = LoadPreferences(p)
	if e != nil || got.Theme != "system" || got.FontSize != 14 || got.DefaultMode != 0 {
		t.Fatalf("invalid preferences: %+v %v", got, e)
	}
}
