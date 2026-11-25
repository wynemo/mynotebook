package notebook

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func put(t *testing.T, p, s string) {
	t.Helper()
	if e := os.WriteFile(p, []byte(s), 0640); e != nil {
		t.Fatal(e)
	}
}
func TestSaveDetectsExternalChange(t *testing.T) {
	p := filepath.Join(t.TempDir(), "中文.md")
	put(t, p, "原稿\r\n")
	d, e := Read(p)
	if e != nil {
		t.Fatal(e)
	}
	v, e := Save(p, "新稿\r\n", d.Version)
	if e != nil {
		t.Fatal(e)
	}
	d, e = Read(p)
	if e != nil || d.Version != v || d.Text != "新稿\r\n" {
		t.Fatalf("round trip: %+v %v", d, e)
	}
	put(t, p, "外部修改")
	if _, e = Save(p, "覆盖", v); !errors.Is(e, ErrConflict) {
		t.Fatalf("expected conflict: %v", e)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "外部修改" {
		t.Fatal("external edits lost")
	}
	if _, e = Save(p, "新建覆盖", ""); !errors.Is(e, ErrConflict) {
		t.Fatalf("new file overwrote existing: %v", e)
	}
	os.Remove(p)
	if _, e = Save(p, "重新创建", v); !errors.Is(e, ErrConflict) {
		t.Fatalf("missing file: %v", e)
	}
}
func TestTreeSearchBoundariesAndUnicode(t *testing.T) {
	dir := t.TempDir()
	os.Mkdir(filepath.Join(dir, "nested"), 0755)
	os.Mkdir(filepath.Join(dir, ".hidden"), 0755)
	put(t, filepath.Join(dir, "note.md"), "开头\n你好 World\n再见 world")
	put(t, filepath.Join(dir, "nested", "other.MD"), "WORLD")
	put(t, filepath.Join(dir, ".hidden", "secret.md"), "world")
	put(t, filepath.Join(dir, "skip.txt"), "world")
	n, e := Root(dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(n.Children) != 2 || !n.Children[0].Dir {
		t.Fatalf("tree: %+v", n.Children)
	}
	r, e := Search(context.Background(), []string{dir, filepath.Join(dir, "nested")}, "world", 20)
	if e != nil || len(r.Matches) != 3 {
		t.Fatalf("search: %+v %v", r, e)
	}
	found := false
	for _, m := range r.Matches {
		if m.Line == 2 {
			found = true
			if m.Start != len("开头\n") {
				t.Fatalf("offset %d", m.Start)
			}
		}
	}
	if !found {
		t.Fatal("unicode line missing")
	}
	r, e = Search(context.Background(), []string{dir}, "world", 1)
	if e != nil || len(r.Matches) != 1 || !r.Truncated {
		t.Fatalf("limit: %+v %v", r, e)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e = Search(ctx, []string{dir}, "world", 20); !errors.Is(e, context.Canceled) {
		t.Fatalf("cancellation: %v", e)
	}
}
func TestImageAttachmentsPortableAndDeduplicated(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "note.md")
	put(t, p, "")
	var b bytes.Buffer
	im := image.NewRGBA(image.Rect(0, 0, 4, 4))
	im.Set(0, 0, color.White)
	png.Encode(&b, im)
	rel, e := AttachImage(p, b.Bytes())
	if e != nil {
		t.Fatal(e)
	}
	again, e := AttachImage(p, b.Bytes())
	if e != nil || again != rel {
		t.Fatalf("dedupe %q %v", again, e)
	}
	got, e := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
	if e != nil || !bytes.Equal(got, b.Bytes()) {
		t.Fatal("image differs")
	}
	if _, e = AttachImage("", b.Bytes()); e == nil {
		t.Fatal("unsaved document accepted")
	}
	if _, e = AttachImage(p, []byte("bad png")); e == nil {
		t.Fatal("bad image accepted")
	}
}
func TestSymlinksNotFollowed(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	p := filepath.Join(dir, "linked.md")
	target := filepath.Join(outside, "note.md")
	put(t, target, "protected")
	if e := os.Symlink(target, p); e != nil {
		t.Skip(e)
	}
	if _, e := Save(p, "overwrite", Digest([]byte("protected"))); e == nil {
		t.Fatal("saved through symlink")
	}
	os.Symlink(outside, filepath.Join(dir, ".assets"))
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if _, e := AttachImage(filepath.Join(dir, "note.md"), b.Bytes()); e == nil {
		t.Fatal("attachment escaped through symlink")
	}
	n, e := Root(dir)
	if e != nil {
		t.Fatal(e)
	}
	if len(n.Children) != 0 {
		t.Fatal("symlink appeared in tree")
	}
}
func TestSessionRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "settings", "workspace.json")
	want := Session{Roots: []string{"/笔记"}, LastFile: "/笔记/a.md", SidebarWidth: 305}
	if e := SaveSession(p, want); e != nil {
		t.Fatal(e)
	}
	got, e := LoadSession(p)
	if e != nil || got.LastFile != want.LastFile || got.SidebarWidth != 305 || len(got.Roots) != 1 {
		t.Fatalf("session %+v %v", got, e)
	}
}
