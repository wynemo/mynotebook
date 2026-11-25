package main

import (
	"bytes"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func testApp() *app {
	return &app{sidebar: true, sidebarWidth: 270, splitPercent: 50, status: "本地 Markdown · Native UI"}
}
func TestNativeEditPreviewSaveAndCloseGuard(t *testing.T) {
	a := testApp()
	tt := ui.NewTester(a.view, 1180, 780)
	if e := tt.Click("创建第一篇笔记"); e != nil {
		t.Fatal(e)
	}
	if e := tt.Click("Markdown 编辑器"); e != nil {
		t.Fatal(e)
	}
	tt.Type("# 我的笔记\n\n你好 **世界**\n")
	n := a.current()
	if n == nil || !strings.Contains(n.text, "你好") || !n.dirty() {
		t.Fatalf("edit failed: %+v", n)
	}
	if e := tt.Click("预览"); e != nil {
		t.Fatal(e)
	}
	if !tt.HasText("我的笔记") {
		t.Fatalf("native preview: %q", tt.Texts())
	}
	n.path = filepath.Join(t.TempDir(), "note.md")
	a.save(n, nil)
	if n.dirty() || a.errorText != "" {
		t.Fatalf("save: %s", a.errorText)
	}
	n.text += "unsaved"
	a.guard(n, func() { a.removeNote(n) })
	tt.Frame()
	if !tt.HasText("保存更改？") {
		t.Fatal("missing save guard")
	}
	if e := tt.Click("取消"); e != nil {
		t.Fatal(e)
	}
	if len(a.notes) != 1 || !n.dirty() {
		t.Fatal("cancel lost changes")
	}
	a.guard(n, func() { a.removeNote(n) })
	tt.Frame()
	if e := tt.Click("不保存"); e != nil {
		t.Fatal(e)
	}
	if len(a.notes) != 0 {
		t.Fatal("discard did not close")
	}
}
func TestNativeFoldersTabsAndSidebar(t *testing.T) {
	a := testApp()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "hello.md"), []byte("# Hello\n"), 0644)
	a.addRoot(dir)
	tt := ui.NewTester(a.view, 1180, 780)
	if e := tt.Click("hello.md"); e != nil {
		t.Fatal(e)
	}
	if a.current() == nil {
		t.Fatal("file not opened")
	}
	if e := tt.Click("新建笔记"); e != nil {
		t.Fatal(e)
	}
	if len(a.notes) != 2 {
		t.Fatal("new tab missing")
	}
	if e := tt.Click("侧栏"); e != nil {
		t.Fatal(e)
	}
	if a.sidebar {
		t.Fatal("sidebar did not collapse")
	}
	if e := tt.Click("侧栏"); e != nil {
		t.Fatal(e)
	}
	if !a.sidebar {
		t.Fatal("sidebar did not expand")
	}
}

// Opt-in screenshot artifacts let developers inspect the actual native view.
func TestNativeScreenshots(t *testing.T) {
	dir := os.Getenv("MYNOTEBOOK_SCREENSHOTS")
	if dir == "" {
		t.Skip("set MYNOTEBOOK_SCREENSHOTS to render native screenshots")
	}
	os.MkdirAll(dir, 0755)
	a := testApp()
	a.newNote()
	a.current().text = "# 每一天，都值得记录\n\n这是一个完全使用 **MyGo Native UI** 构建的 Markdown 笔记本。\n\n## 今天的计划\n\n- [x] 整理思路\n- [ ] 写下新的想法\n\n> 让文字回到文字，让注意力回到当下。\n\n| 功能 | 状态 |\n| --- | --- |\n| 原生编辑 | 已就绪 |\n| 本地附件 | 已就绪 |\n\n```go\nfmt.Println(\"Hello, notebook\")\n```\n"
	a.current().saved = a.current().text
	a.addRoot(".")
	a.mode = 2
	for _, dark := range []bool{false, true} {
		tt := ui.NewTester(a.view, 1180, 780)
		tt.SetDark(dark)
		name := "light.png"
		if dark {
			name = "dark.png"
		}
		f, e := os.Create(filepath.Join(dir, name))
		if e != nil {
			t.Fatal(e)
		}
		e = png.Encode(f, tt.Image())
		f.Close()
		if e != nil {
			t.Fatal(e)
		}
	}
}

func TestNativeImageInsertionAtSelectionAndUndo(t *testing.T) {
	a := testApp()
	a.newNote()
	n := a.current()
	n.path = filepath.Join(t.TempDir(), "image-note.md")
	n.text = "你好，世界"
	n.saved = n.text
	tt := ui.NewTester(a.view, 1180, 780)
	if e := tt.Click("Markdown 编辑器"); e != nil {
		t.Fatal(e)
	}
	// Select the source with the platform's standard editing command.
	tt.Command("selectAll")
	var imageData bytes.Buffer
	if e := png.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 2, 2))); e != nil {
		t.Fatal(e)
	}
	a.attachImage(n, imageData.Bytes())
	tt.Frame()
	if !strings.HasPrefix(n.text, "![图片](.assets/image-") || strings.Contains(n.text, "你好") {
		t.Fatalf("paste did not replace selection: %q", n.text)
	}
	tt.Command("undo")
	if n.text != "你好，世界" {
		t.Fatalf("undo lost original source: %q", n.text)
	}
}
func TestSaveConflictKeepsDraftOpen(t *testing.T) {
	a := testApp()
	dir := t.TempDir()
	p := filepath.Join(dir, "note.md")
	os.WriteFile(p, []byte("original"), 0644)
	a.openFile(p)
	n := a.current()
	n.text = "my draft"
	os.WriteFile(p, []byte("external"), 0644)
	closed := false
	a.save(n, func() { closed = true })
	if closed || n.text != "my draft" || !n.dirty() || a.errorText == "" {
		t.Fatal("failed save discarded the draft")
	}
	b, _ := os.ReadFile(p)
	if string(b) != "external" {
		t.Fatal("external change overwritten")
	}
}
