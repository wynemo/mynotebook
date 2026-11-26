package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"mynotebook/internal/notebook"
	"os"
	"path/filepath"
	"testing"
)

func writeNote(t *testing.T, path, text string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}
func TestMoveAndRenameOpenNotes(t *testing.T) {
	a := testApp()
	root, err := notebook.Canonical(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	src := filepath.Join(root, "source")
	dst := filepath.Join(root, "target")
	os.Mkdir(src, 0755)
	os.Mkdir(dst, 0755)
	path := filepath.Join(src, "note.md")
	writeNote(t, path, "saved")
	a.addRoot(root)
	a.openFile(path)
	n := a.current()
	n.text = "unsaved"
	if err := a.movePath(src, filepath.Join(dst, "source")); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dst, "source", "note.md")
	if n.path != want || n.text != "unsaved" || !n.dirty() {
		t.Fatal("move lost open buffer")
	}
	collision := filepath.Join(dst, "source", "exists.md")
	writeNote(t, collision, "keep")
	if err := a.movePath(want, collision); err == nil {
		t.Fatal("overwrote existing file")
	}
	data, _ := os.ReadFile(collision)
	if string(data) != "keep" {
		t.Fatal("overwritten")
	}
	if err := a.movePath(dst, filepath.Join(dst, "source", "recursive")); err == nil {
		t.Fatal("recursive move accepted")
	}
	a.renamePath = want
	a.renameName = "改名"
	a.renameOpen = true
	a.renameEntry()
	if a.renameOpen || n.name() != "改名.md" {
		t.Fatalf("rename failed: %s", a.renameError)
	}
	a.save(n, nil)
	if a.errorText != "" || n.dirty() {
		t.Fatal("save after move failed", a.errorText)
	}
}
func TestTabMenusGuardAndReopen(t *testing.T) {
	a := testApp()
	dir, err := notebook.Canonical(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.md", "b.md", "c.md"} {
		path := filepath.Join(dir, name)
		writeNote(t, path, name)
		a.openFile(path)
	}
	tt := ui.NewTester(a.view, 1180, 780)
	a.notes[1].text = "dirty"
	if err := tt.RightClick("a.md"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("关闭右侧标签"); err != nil {
		t.Fatal(err)
	}
	if !a.ask {
		t.Fatal("missing dirty guard")
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if len(a.notes) != 3 {
		t.Fatal("cancel closed tabs")
	}
	if err := tt.RightClick("a.md"); err != nil {
		t.Fatal(err)
	}
	if err := tt.ChooseMenuItem("关闭其他标签"); err != nil {
		t.Fatal(err)
	}
	if err := tt.Click("不保存"); err != nil {
		t.Fatal(err)
	}
	if len(a.notes) != 1 || len(a.closed) != 2 {
		t.Fatal("bulk close failed")
	}
	a.reopenNote()
	tt.Frame()
	if len(a.notes) != 2 || a.current().name() != "c.md" {
		t.Fatal("reopen failed")
	}
	a.reopenNote()
	tt.Frame()
	if a.current().text != "dirty" {
		t.Fatal("closed draft content lost")
	}
}
func TestExternalChangesPreserveDirtyBuffers(t *testing.T) {
	a := testApp()
	dir, err := notebook.Canonical(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "a.md")
	writeNote(t, path, "old")
	a.addRoot(dir)
	a.openFile(path)
	n := a.current()
	writeNote(t, path, "external")
	a.applyExternal(dir, nil)
	if n.text != "external" || n.dirty() {
		t.Fatal("clean note not reloaded")
	}
	n.text = "local edit"
	writeNote(t, path, "another external edit")
	a.applyExternal(dir, nil)
	if n.text != "local edit" || n.external == "" {
		t.Fatal("dirty edit overwritten")
	}
	a.save(n, nil)
	if a.errorText == "" {
		t.Fatal("conflicting save accepted")
	}
	a.errorText = ""
	renamed := filepath.Join(dir, "b.md")
	os.Rename(path, renamed)
	a.applyExternal(dir, []mygo.FileEvent{{Op: mygo.FileRename, OldPath: "a.md", Path: "b.md"}})
	if n.path != renamed || n.text != "local edit" {
		t.Fatal("external rename lost edit")
	}
	os.Remove(renamed)
	a.applyExternal(dir, nil)
	if n.external == "" || n.text != "local edit" {
		t.Fatal("external delete lost edit")
	}
	if len(a.roots[0].Children) != 0 {
		t.Fatal("tree not refreshed")
	}
}
func TestFindReplaceUnicodeAndUndo(t *testing.T) {
	a := testApp()
	a.newNote()
	n := a.current()
	n.text = "你好 世界，你好"
	n.saved = n.text
	tt := ui.NewTester(a.view, 1180, 780)
	a.openFind()
	a.findText = "你好"
	a.findNext(n)
	tt.Frame()
	if n.start != 0 || n.end != 2 {
		t.Fatalf("unicode selection %d:%d", n.start, n.end)
	}
	a.replaceText = "您好"
	a.replaceOne(n)
	tt.Frame()
	if n.text != "您好 世界，你好" {
		t.Fatalf("replace: %q", n.text)
	}
	tt.Command("undo")
	if n.text != n.saved {
		t.Fatalf("replacement undo: %q", n.text)
	}
	tt.Command("redo")
	if n.text != "您好 世界，你好" {
		t.Fatal("redo failed")
	}
	a.findText = "你好"
	a.findNext(n)
	tt.Frame()
	if n.start != 6 || n.end != 8 {
		t.Fatalf("next selection: %d:%d", n.start, n.end)
	}
	a.beginLink(false)
	a.linkURL = "https://example.com/a b"
	a.insertLink()
	tt.Frame()
	if n.text != "您好 世界，[你好](https://example.com/a%20b)" {
		t.Fatalf("insert link: %q", n.text)
	}
	tt.Command("undo")
	if n.text != "您好 世界，你好" {
		t.Fatal("link undo failed")
	}
	a.openFind()
	a.findText = "好"
	a.replaceText = "妙"
	tt.Frame()
	if err := tt.Click("全部替换"); err != nil {
		t.Fatal(err)
	}
	if n.text != "您妙 世界，你妙" {
		t.Fatalf("replace all: %q", n.text)
	}
	tt.Command("undo")
	if n.text != "您好 世界，你好" {
		t.Fatal("replace all undo failed")
	}
}

func TestTreeDragMovesIntoFolder(t *testing.T) {
	a := testApp()
	a.addRoot(t.TempDir())
	dir := a.roots[0].Path
	os.Mkdir(filepath.Join(dir, "folder"), 0755)
	path := filepath.Join(dir, "drag.md")
	writeNote(t, path, "drag me")
	a.refresh()
	a.openFile(path)
	tt := ui.NewTester(a.view, 1180, 780)
	// The first match is the tree row, before the tab with the same name.
	src, ok := tt.Find("drag.md")
	if !ok {
		t.Fatal("missing source")
	}
	dst, ok := tt.Find("folder")
	if !ok {
		t.Fatal("missing target")
	}
	tt.Press(src.X+12, src.Y+src.H/2)
	tt.Move(src.X+30, src.Y+src.H/2+8)
	tt.Move(dst.X+20, dst.Y+dst.H/2)
	tt.Release(dst.X+20, dst.Y+dst.H/2)
	if _, err := os.Stat(filepath.Join(dir, "folder", "drag.md")); err != nil {
		t.Fatal("drop did not move file:", err, a.errorText)
	}
	if a.current().path != filepath.Join(dir, "folder", "drag.md") {
		t.Fatal("drop did not update tab")
	}
}
func TestTrashGuardFailureAndSuccess(t *testing.T) {
	a := testApp()
	a.addRoot(t.TempDir())
	path := filepath.Join(a.roots[0].Path, "trash.md")
	writeNote(t, path, "saved")
	a.openFile(path)
	n := a.current()
	n.text = "unsaved"
	calls := 0
	trash := func(path string) error { calls++; return os.Remove(path) }
	tt := ui.NewTester(a.view, 1180, 780)
	a.trashPath(path, trash)
	tt.Frame()
	if calls != 0 || !a.ask {
		t.Fatal("trash ran before dirty guard")
	}
	if err := tt.Click("取消"); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || len(a.notes) != 1 {
		t.Fatal("cancel trashed note")
	}
	n.text = n.saved
	a.trashPath(path, func(string) error { return os.ErrPermission })
	if len(a.notes) != 1 || a.errorText == "" {
		t.Fatal("failed trash closed tab")
	}
	a.errorText = ""
	a.trashPath(path, trash)
	if calls != 1 || len(a.notes) != 0 {
		t.Fatal("trash failed")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("not removed by trash adapter")
	}
}
