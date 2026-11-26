package notebook

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveNeverReplaces(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "src"), filepath.Join(dir, "dst")
	os.WriteFile(src, []byte("source"), 0644)
	os.WriteFile(dst, []byte("destination"), 0644)
	if err := Move(src, dst); err == nil {
		t.Fatal("replaced an existing destination")
	}
	data, _ := os.ReadFile(dst)
	if string(data) != "destination" {
		t.Fatal("destination changed")
	}
	if err := Move(src, filepath.Join(dir, "moved")); err != nil {
		t.Fatal(err)
	}
	os.Mkdir(src, 0755)
	os.Mkdir(filepath.Join(dir, "existing-dir"), 0755)
	if err := Move(src, filepath.Join(dir, "existing-dir")); err == nil {
		t.Fatal("replaced an existing directory")
	}
}
