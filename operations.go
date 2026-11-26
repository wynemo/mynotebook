package main

import (
	"context"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"mynotebook/internal/notebook"
	"os"
	"path/filepath"
	"strings"
)

func within(parent, path string) bool {
	rel, err := filepath.Rel(parent, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (a *app) fileMenu(m *ui.Menu, n *notebook.Node) {
	parent := n.Path
	if !n.Dir {
		parent = filepath.Dir(parent)
	}
	a.creationMenu(m, parent)
	m.Separator()
	if m.Item("重命名…").Chosen() {
		a.renamePath, a.renameName, a.renameError = n.Path, n.Name, ""
		a.renameOpen = true
		a.renameFocus.Focus()
	}
	if m.Item("移动到…").Disabled(a.busy).Chosen() {
		a.chooseMove(n.Path)
	}
	if m.Item("在系统文件夹中显示").Chosen() {
		mygo.Shell.ShowItemInFolder(n.Path)
	}
	if m.Item("删除到废纸篓").Chosen() {
		a.trashPath(n.Path, mygo.Shell.TrashItem)
	}
}

func (a *app) guardPaths(path string, done func()) {
	var tabs []*note
	for _, n := range a.notes {
		if n.path != "" && within(path, n.path) {
			tabs = append(tabs, n)
		}
	}
	a.guardTabs(tabs, done)
}
func (a *app) guardTabs(tabs []*note, done func()) {
	if len(tabs) == 0 {
		done()
		return
	}
	a.guard(tabs[0], func() { a.guardTabs(tabs[1:], done) })
}
func (a *app) chooseMove(path string) {
	if a.win == nil || a.busy {
		return
	}
	a.busy = true
	go func() {
		dirs, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "移动到文件夹", Directory: true})
		a.win.Update(func() {
			a.busy = false
			if err != nil {
				a.fail(err)
				return
			}
			if len(dirs) > 0 {
				a.fail(a.movePath(path, filepath.Join(dirs[0], filepath.Base(path))))
			}
		})
	}()
}
func (a *app) movePath(src, dst string) error {
	src, err := notebook.Canonical(src)
	if err != nil {
		return err
	}
	parent, err := notebook.Canonical(filepath.Dir(dst))
	if err != nil {
		return err
	}
	dst = filepath.Join(parent, filepath.Base(dst))
	if src == dst {
		return nil
	}
	if within(src, dst) {
		return fmt.Errorf("不能移动到自身或自己的子目录")
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("目标已存在，未覆盖")
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := notebook.Move(src, dst); err != nil {
		return fmt.Errorf("移动失败（跨磁盘移动暂不支持）：%w", err)
	}
	a.remapPaths(src, dst)
	a.refresh()
	a.startSearch()
	a.persist()
	a.syncWatchers()
	return nil
}
func (a *app) remapPaths(src, dst string) {
	remap := func(p string) string {
		if p == "" || !within(src, p) {
			return p
		}
		rel, _ := filepath.Rel(src, p)
		return filepath.Join(dst, rel)
	}
	for _, n := range append(append([]*note(nil), a.notes...), a.closed...) {
		p := remap(n.path)
		if p != n.path {
			n.path = p
			a.configurePreview(n)
		}
	}
	var walk func(*notebook.Node)
	walk = func(n *notebook.Node) {
		n.Path = remap(n.Path)
		n.Name = filepath.Base(n.Path)
		for _, child := range n.Children {
			walk(child)
		}
	}
	for _, r := range a.roots {
		walk(r)
	}
}
func (a *app) fileRow(c *ui.Context, e ui.Element, n *notebook.Node) {
	e.ContextMenu(func(m *ui.Menu) { a.fileMenu(m, n) }).Drag(n.Path)
	if !n.Dir {
		return
	}
	if _, ok := ui.DragOver[string](e); ok {
		e.Background(paper(c).selected)
	}
	if src, ok := ui.Drop[string](e); ok {
		a.fail(a.movePath(src, filepath.Join(n.Path, filepath.Base(src))))
	}
}

func (a *app) closeTabs(tabs []*note) {
	if len(tabs) == 0 {
		return
	}
	n := tabs[0]
	a.guard(n, func() { a.removeNote(n); a.closeTabs(tabs[1:]) })
}
func (a *app) tabMenu(m *ui.Menu, n *note) {
	if m.Item("关闭标签").Chosen() {
		a.closeTabs([]*note{n})
	}
	if m.Item("关闭其他标签").Chosen() {
		var tabs []*note
		for _, tab := range a.notes {
			if tab != n {
				tabs = append(tabs, tab)
			}
		}
		a.closeTabs(tabs)
	}
	if m.Item("关闭右侧标签").Chosen() {
		for i, tab := range a.notes {
			if tab == n {
				a.closeTabs(append([]*note(nil), a.notes[i+1:]...))
				break
			}
		}
	}
	m.Separator()
	if m.Item("重新打开关闭的笔记").Disabled(len(a.closed) == 0).Chosen() {
		a.reopenNote()
	}
}
func (a *app) reopenNote() {
	if len(a.closed) == 0 {
		return
	}
	n := a.closed[len(a.closed)-1]
	a.closed = a.closed[:len(a.closed)-1]
	if n.path != "" {
		for _, tab := range a.notes {
			if tab.path == n.path {
				a.active = tab.id
				return
			}
		}
		if !n.dirty() {
			if d, err := notebook.Read(n.path); err == nil {
				n.text, n.saved, n.version = d.Text, d.Text, d.Version
			}
		}
	}
	a.nextID++
	n.id = a.nextID
	n.jump = true
	a.notes = append(a.notes, n)
	a.active = n.id
	a.settingsOpen = false
	a.persist()
	a.syncWatchers()
}

// Watch callbacks only change application state on the window's UI thread.
func (a *app) syncWatchers() {
	if a.win == nil {
		return
	}
	if a.watchers == nil {
		a.watchers = map[string]context.CancelFunc{}
	}
	wanted := map[string]bool{}
	for _, r := range a.roots {
		wanted[r.Path] = true
	}
	for _, n := range a.notes {
		if n.path != "" {
			covered := false
			for _, r := range a.roots {
				if within(r.Path, n.path) {
					covered = true
					break
				}
			}
			if !covered {
				wanted[filepath.Dir(n.path)] = true
			}
		}
	}
	for path, cancel := range a.watchers {
		if !wanted[path] {
			cancel()
			delete(a.watchers, path)
		}
	}
	for path := range wanted {
		if a.watchers[path] != nil {
			continue
		}
		ctx, cancel := context.WithCancel(context.Background())
		w, err := mygo.WatchFiles(ctx, path, mygo.FileWatchOptions{Recursive: true, Ignore: func(p string, dir bool) bool {
			name := filepath.Base(p)
			return strings.HasPrefix(name, ".") || name == "node_modules"
		}})
		if err != nil {
			cancel()
			a.fail(err)
			continue
		}
		a.watchers[path] = cancel
		go func() {
			defer w.Close()
			for {
				events, err := w.Next(ctx)
				if err != nil {
					if ctx.Err() == nil {
						a.win.Update(func() { a.fail(err) })
					}
					return
				}
				a.win.Update(func() {
					if ctx.Err() == nil {
						a.applyExternal(path, events)
					}
				})
			}
		}()
	}
}
func (a *app) stopWatchers() {
	for _, cancel := range a.watchers {
		cancel()
	}
	a.watchers = nil
}
func (a *app) applyExternal(root string, events []mygo.FileEvent) {
	for _, ev := range events {
		if ev.Op == mygo.FileRename {
			old := filepath.Join(root, filepath.FromSlash(ev.OldPath))
			dst := filepath.Join(root, filepath.FromSlash(ev.Path))
			if _, err := os.Stat(old); os.IsNotExist(err) {
				a.remapPaths(old, dst)
			}
		}
	}
	for _, n := range a.notes {
		if n.path == "" {
			continue
		}
		d, err := notebook.Read(n.path)
		if err != nil {
			if os.IsNotExist(err) {
				n.external = "文件已被外部删除或移动；当前内容仍保留，可另存为。"
			} else {
				n.external = err.Error()
			}
			continue
		}
		if d.Version == n.version {
			n.external = ""
			continue
		}
		if n.dirty() {
			n.external = "文件已被外部修改；当前未保存的内容已保留。请另存为或重新载入。"
			continue
		}
		n.text, n.saved, n.version = d.Text, d.Text, d.Version
		n.external = ""
		a.configurePreview(n)
	}
	a.refresh()
	a.startSearch()
	a.persist()
	a.syncWatchers()
}

func (a *app) trashPath(path string, trash func(string) error) {
	a.guardPaths(path, func() {
		if err := trash(path); err != nil {
			a.fail(err)
			return
		}
		for _, tab := range append([]*note(nil), a.notes...) {
			if tab.path != "" && within(path, tab.path) {
				a.removeNote(tab)
			}
		}
		for _, r := range append([]*notebook.Node(nil), a.roots...) {
			if within(path, r.Path) {
				a.removeRoot(r)
			}
		}
		a.refresh()
		a.startSearch()
		a.syncWatchers()
	})
}
