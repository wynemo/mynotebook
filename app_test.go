package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"mynotebook/internal/notebook"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func testApp() *app {
	return &app{sidebar: true, sidebarWidth: 270, splitPercent: 50, status: "随时写下新的想法"}
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
	root := filepath.Join(t.TempDir(), "个人笔记")
	project := filepath.Join(filepath.Dir(root), "项目文档")
	for _, p := range []string{root, project, filepath.Join(root, "日记"), filepath.Join(root, "读书摘录")} {
		if e := os.MkdirAll(p, 0755); e != nil {
			t.Fatal(e)
		}
	}
	sample := "# 每一天，都值得记录\n\n不用急着找到答案。先把今天的想法写下来。\n\n## 今天的小计划\n\n- [x] 留出一段不被打扰的时间\n- [ ] 整理最近读过的书\n- [ ] 给明天留一个新的问题\n\n> 慢慢来，文字会替你记住。\n\n## 值得留下的片刻\n\n阳光落在桌角，咖啡还温着。\n一个普通的下午，也可以是一篇笔记的开始。\n"
	for name, body := range map[string]string{"每一天，都值得记录.md": sample, "灵感收集.md": "# 灵感收集\n\n收集路上遇到的好想法。", "写作清单.md": "# 写作清单\n\n- 随笔\n- 阅读笔记", "日记/十一月.md": "# 十一月\n", "读书摘录/阅读中的句子.md": "# 阅读中的句子\n"} {
		if e := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), []byte(body), 0644); e != nil {
			t.Fatal(e)
		}
	}
	os.WriteFile(filepath.Join(project, "产品想法.md"), []byte("# 产品想法\n"), 0644)
	a := testApp()
	a.addRoot(root)
	a.addRoot(project)
	for _, name := range []string{"灵感收集.md", "写作清单.md", "每一天，都值得记录.md"} {
		a.openFile(filepath.Join(root, name))
	}
	a.mode = 2
	render := func(name string, w, h int, dark bool) {
		if n := a.current(); n != nil {
			n.jump = true
			n.start = 0
			n.end = 0
		}
		tt := ui.NewTester(a.view, w, h)
		tt.SetDark(dark)
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
	render("light.png", 1180, 780, false)
	render("dark.png", 1180, 780, true)
	render("compact.png", 760, 560, false)
	a.searchOpen = true
	a.query = "想法"
	a.results, _ = notebook.Search(context.Background(), a.paths(), a.query, 500)
	render("search.png", 1180, 780, false)
	a.searchOpen = false
	a.notes = nil
	a.active = 0
	render("welcome.png", 1180, 780, false)
	render("welcome-dark.png", 1180, 780, true)
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

func TestPaperSearchNavigationAndViewModes(t *testing.T) {
	a := testApp()
	root := t.TempDir()
	path := filepath.Join(root, "想法.md")
	if e := os.WriteFile(path, []byte("# 想法\n找到这一行\n"), 0644); e != nil {
		t.Fatal(e)
	}
	a.addRoot(root)
	tt := ui.NewTester(a.view, 1180, 780)
	if e := tt.Click("搜索"); e != nil {
		t.Fatal(e)
	}
	if !a.searchOpen || !tt.Focused("全局搜索") {
		t.Fatal("search field did not receive focus")
	}
	a.query = "找到"
	a.results, _ = notebook.Search(context.Background(), a.paths(), a.query, 500)
	tt.Frame()
	if e := tt.Click("想法.md : 2"); e != nil {
		t.Fatal(e)
	}
	canonical, _ := notebook.Canonical(path)
	if a.current() == nil || a.current().path != canonical || a.current().start != 5 {
		t.Fatalf("search navigation: %+v", a.current())
	}
	for i, label := range []string{"预览", "分屏", "编辑"} {
		if e := tt.Click(label); e != nil {
			t.Fatal(e)
		}
		expected := []int{1, 2, 0}[i]
		if a.mode != expected {
			t.Fatalf("%s: mode %d", label, a.mode)
		}
	}
}
