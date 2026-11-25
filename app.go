package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"mynotebook/internal/notebook"
)

type note struct {
	id                         int
	path, text, saved, version string
	insert                     string
	start, end                 int
	jump                       bool
	scroll                     ui.ScrollState
	md                         *markdown.Options
}

func (n *note) dirty() bool { return n.text != n.saved }
func (n *note) name() string {
	if n.path == "" {
		return "未命名.md"
	}
	return filepath.Base(n.path)
}

type app struct {
	win                        *mygo.Window
	config                     string
	roots                      []*notebook.Node
	notes                      []*note
	active, nextID             int
	sidebar                    bool
	sidebarWidth, splitPercent float32
	mode                       int
	query                      string
	results                    notebook.SearchResult
	searching                  bool
	searchID                   int
	cancelSearch               context.CancelFunc
	searchFocus                ui.Handle
	searchOpen                 bool
	busy                       bool
	status                     string
	errorText                  string
	ask                        bool
	pending                    *note
	after                      func()
}

func newApp(config string) *app {
	a := &app{config: config, sidebar: true, sidebarWidth: 270, splitPercent: 50, status: "随时写下新的想法"}
	s, e := notebook.LoadSession(config)
	if e != nil {
		a.fail(e)
	} else {
		a.sidebarWidth = s.SidebarWidth
		for _, p := range s.Roots {
			if n, e := notebook.Root(p); e == nil {
				a.roots = append(a.roots, n)
			} else {
				a.fail(e)
			}
		}
		if s.LastFile != "" {
			a.openFile(s.LastFile)
		}
	}
	return a
}
func (a *app) current() *note {
	for _, n := range a.notes {
		if n.id == a.active {
			return n
		}
	}
	return nil
}
func (a *app) fail(e error) {
	if e != nil {
		a.errorText = e.Error()
	}
}
func (a *app) paths() []string {
	p := make([]string, 0, len(a.roots))
	for _, r := range a.roots {
		p = append(p, r.Path)
	}
	return p
}
func (a *app) persist() {
	if a.config == "" {
		return
	}
	last := ""
	if n := a.current(); n != nil {
		last = n.path
	}
	a.fail(notebook.SaveSession(a.config, notebook.Session{Roots: a.paths(), LastFile: last, SidebarWidth: a.sidebarWidth}))
}
func (a *app) addRoot(path string) {
	n, e := notebook.Root(path)
	if e != nil {
		a.fail(e)
		return
	}
	for _, r := range a.roots {
		if r.Path == n.Path {
			return
		}
	}
	a.roots = append(a.roots, n)
	a.persist()
	a.startSearch()
}
func (a *app) chooseFolders() {
	if a.busy || a.win == nil {
		return
	}
	a.busy = true
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "添加笔记文件夹", Directory: true, Multiple: true})
		a.win.Update(func() {
			a.busy = false
			a.fail(e)
			for _, p := range paths {
				a.addRoot(p)
			}
		})
	}()
}
func (a *app) chooseFile() {
	if a.busy || a.win == nil {
		return
	}
	a.busy = true
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "打开 Markdown", Filters: []mygo.FileFilter{{Name: "Markdown", Extensions: []string{"md", "markdown", "mdown"}}}})
		a.win.Update(func() {
			a.busy = false
			a.fail(e)
			for _, p := range paths {
				a.openFile(p)
			}
		})
	}()
}
func (a *app) newNote() {
	a.nextID++
	n := &note{id: a.nextID, md: &markdown.Options{}, jump: true}
	a.notes = append(a.notes, n)
	a.active = n.id
	a.mode = 0
}
func (a *app) openFile(path string) {
	p, e := notebook.Canonical(path)
	if e != nil {
		a.fail(e)
		return
	}
	for _, n := range a.notes {
		if n.path == p {
			a.active = n.id
			return
		}
	}
	d, e := notebook.Read(p)
	if e != nil {
		a.fail(e)
		return
	}
	a.nextID++
	n := &note{id: a.nextID, path: d.Path, text: d.Text, saved: d.Text, version: d.Version, jump: true}
	a.configurePreview(n)
	a.notes = append(a.notes, n)
	a.active = n.id
	a.persist()
}
func (a *app) configurePreview(n *note) {
	n.md = &markdown.Options{LoadImage: markdown.FileImages(filepath.Dir(n.path))}
	n.md.OnLink = func(dest string) {
		if strings.HasPrefix(dest, "#") {
			n.md.Anchor = dest
			n.md.AnchorVersion++
			return
		}
		u, e := url.Parse(dest)
		if e != nil {
			a.fail(e)
			return
		}
		if u.Scheme == "http" || u.Scheme == "https" || u.Scheme == "mailto" {
			a.fail(mygo.Shell.OpenExternal(dest))
			return
		}
		if u.Scheme == "" && notebook.IsMarkdown(u.Path) {
			path := u.Path
			if !filepath.IsAbs(path) {
				path = filepath.Join(filepath.Dir(n.path), filepath.FromSlash(path))
			}
			a.openFile(path)
			if cur := a.current(); cur != nil {
				cur.md.Anchor = u.Fragment
			}
		}
	}
}
func (a *app) save(n *note, done func()) {
	if n == nil {
		return
	}
	if n.path == "" {
		a.saveAs(n, done)
		return
	}
	v, e := notebook.Save(n.path, n.text, n.version)
	if e != nil {
		a.fail(e)
		return
	}
	n.version = v
	n.saved = n.text
	a.status = "已保存 · " + n.name()
	if done != nil {
		done()
	}
}
func (a *app) saveAs(n *note, done func()) {
	if n == nil || a.busy || a.win == nil {
		return
	}
	a.busy = true
	initial := n.path
	if initial == "" {
		initial = n.name()
		if len(a.roots) > 0 {
			initial = filepath.Join(a.roots[0].Path, initial)
		}
	}
	go func() {
		path, e := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, Title: "保存 Markdown", DefaultPath: initial, Filters: []mygo.FileFilter{{Name: "Markdown", Extensions: []string{"md"}}}})
		a.win.Update(func() {
			a.busy = false
			if e != nil {
				a.fail(e)
				return
			}
			if path == "" {
				return
			}
			if !notebook.IsMarkdown(path) {
				path += ".md"
			}
			// The native Save dialog confirms replacement. Still compare its latest
			// contents so this explicit Save As does not race an external writer.
			version := ""
			if _, e := os.Stat(path); e == nil {
				d, e := notebook.Read(path)
				if e != nil {
					a.fail(e)
					return
				}
				version = d.Version
			}
			for _, other := range a.notes {
				if other != n && other.path == path {
					a.fail(fmt.Errorf("文件已在另一个标签中打开"))
					return
				}
			}
			v, e := notebook.Save(path, n.text, version)
			if e != nil {
				a.fail(e)
				return
			}
			n.path = path
			n.saved = n.text
			n.version = v
			a.configurePreview(n)
			a.status = "已保存 · " + n.name()
			a.refresh()
			a.persist()
			if done != nil {
				done()
			}
		})
	}()
}
func (a *app) refresh() {
	for _, r := range a.roots {
		a.refreshNode(r)
	}
}
func (a *app) refreshNode(n *notebook.Node) {
	if !n.Loaded {
		return
	}
	old := map[string]*notebook.Node{}
	for _, child := range n.Children {
		old[child.Path] = child
	}
	if e := notebook.Load(n); e != nil {
		a.fail(e)
		return
	}
	for i, child := range n.Children {
		if prev := old[child.Path]; prev != nil && prev.Dir == child.Dir {
			n.Children[i] = prev
			a.refreshNode(prev)
		}
	}
}
func (a *app) removeNote(n *note) {
	for i, v := range a.notes {
		if v == n {
			a.notes = append(a.notes[:i], a.notes[i+1:]...)
			break
		}
	}
	if a.active == n.id {
		a.active = 0
		if len(a.notes) > 0 {
			a.active = a.notes[len(a.notes)-1].id
		}
	}
	a.persist()
}
func (a *app) guard(n *note, after func()) {
	if n == nil || !n.dirty() {
		after()
		return
	}
	a.pending = n
	a.after = after
	a.ask = true
}
func (a *app) hasChanges() bool {
	for _, n := range a.notes {
		if n.dirty() {
			return true
		}
	}
	return false
}
func (a *app) requestQuit() {
	if a.ask {
		return
	}
	for _, n := range a.notes {
		if n.dirty() {
			a.active = n.id
			a.guard(n, func() { a.removeNote(n); a.requestQuit() })
			return
		}
	}
	a.persist()
	if a.cancelSearch != nil {
		a.cancelSearch()
	}
	if a.win != nil {
		a.win.Destroy()
		mygo.App.Quit()
	}
}
func (a *app) paste(n *note, text string) bool {
	if text != "" {
		return false
	}
	b := mygo.Clipboard.ReadImage()
	if len(b) == 0 {
		return false
	}
	a.attachImage(n, b)
	return true
}

func (a *app) attachImage(n *note, b []byte) {
	path, e := notebook.AttachImage(n.path, b)
	if e != nil {
		a.fail(e)
		return
	}
	n.insert = "![图片](" + path + ")"
	a.status = "图片已保存至 .assets"
}
func (a *app) startSearch() {
	if a.cancelSearch != nil {
		a.cancelSearch()
	}
	a.searchID++
	id := a.searchID
	if strings.TrimSpace(a.query) == "" {
		a.searching = false
		a.results = notebook.SearchResult{}
		return
	}
	if a.win == nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.cancelSearch = cancel
	a.searching = true
	roots, query := a.paths(), a.query
	go func() {
		r, e := notebook.Search(ctx, roots, query, 500)
		if ctx.Err() != nil {
			return
		}
		a.win.Update(func() {
			if a.searchID != id {
				return
			}
			a.searching = false
			a.results = r
			a.fail(e)
		})
	}()
}
func (a *app) jumpTo(m notebook.Match) {
	a.openFile(m.Path)
	n := a.current()
	if n == nil || n.path != m.Path {
		return
	}
	a.mode = 0
	// Search uses saved files; locate the line in the current buffer again.
	lines := strings.Split(n.text, "\n")
	line := min(m.Line-1, len(lines)-1)
	start := 0
	for _, s := range lines[:line] {
		start += utf8.RuneCountInString(s) + 1
	}
	n.start = start
	n.end = start + utf8.RuneCountInString(lines[line])
	n.jump = true
}
