package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
	"mynotebook/internal/notebook"
)

func (a *app) view(c *ui.Context) {
	mod := ui.Ctrl
	if runtime.GOOS == "darwin" {
		mod = ui.Cmd
	}
	if c.Shortcut(mod, ui.KeyS) {
		a.save(a.current(), nil)
	}
	if c.Shortcut(mod, ui.KeyN) {
		a.newNote()
	}
	if c.Shortcut(mod, ui.KeyB) {
		a.sidebar = !a.sidebar
	}
	if c.Shortcut(mod|ui.Shift, ui.KeyF) {
		a.sidebar = true
		a.searchOpen = true
		a.searchFocus.Focus()
	}
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Height(56).Padding(10, 16).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			ui.Button(c, "侧栏").Tooltip("显示 / 隐藏侧栏 · Cmd/Ctrl+B").OnClick(func() { a.sidebar = !a.sidebar })
			ui.Text(c, "MyNotebook").Bold().FontSize(16)
			ui.Box(c).Grow(1)
			ui.Button(c, "打开文件").Disabled(a.busy).OnClick(a.chooseFile)
			ui.Button(c, "添加文件夹").Disabled(a.busy).OnClick(a.chooseFolders)
			ui.Button(c, "新建笔记").OnClick(a.newNote)
			ui.PrimaryButton(c, "保存").Disabled(a.current() == nil || a.busy).OnClick(func() { a.save(a.current(), nil) })
		})
		if a.sidebar {
			ui.Split(c, &a.sidebarWidth, func() { a.sidebarView(c) }, func() { a.documentView(c) }).Grow(1).FillWidth().SplitReset(270).OnChange(a.persist)
		} else {
			a.documentView(c)
		}
		ui.Row(c).Height(28).Padding(4, 16).Gap(12).Background(t.Surface).Children(func() {
			ui.Text(c, a.status).FontSize(11).TextColor(t.TextMuted).SingleLine().Grow(1)
			if n := a.current(); n != nil {
				state := "已保存"
				if n.dirty() {
					state = "未保存"
				}
				ui.Text(c, fmt.Sprintf("%s · %d 字符 · UTF-8", state, utf8.RuneCountInString(n.text))).FontSize(11).TextColor(t.TextMuted)
			}
		})
	})
	if a.ask && a.pending != nil {
		switch ui.AlertDialog(c, &a.ask, "保存更改？", a.pending.name()+" 有未保存的更改。", "取消", "不保存", "保存") {
		case 0, ui.AlertDismissed:
			a.pending = nil
			a.after = nil
		case 1:
			fn := a.after
			a.after = nil
			a.pending = nil
			if fn != nil {
				fn()
			}
		case 2:
			n, fn := a.pending, a.after
			a.pending = nil
			a.after = nil
			a.save(n, fn)
		}
	}
	if a.errorText != "" {
		show := true
		if ui.AlertDialog(c, &show, "操作未完成", a.errorText, "知道了") >= 0 || !show {
			a.errorText = ""
		}
	}
}
func (a *app) sidebarView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Surface).Padding(12).Gap(12).Children(func() {
		page := 0
		if a.searchOpen {
			page = 1
		}
		if ui.Segmented(c, &page, "文件夹", "搜索").Changed() {
			a.searchOpen = page == 1
			if a.searchOpen {
				a.startSearch()
			}
		}
		if a.searchOpen {
			search := ui.SearchField(c, &a.query).Label("全局搜索").Placeholder("搜索所有文件夹…").Bind(&a.searchFocus)
			if search.Changed() {
				a.startSearch()
			}
			ui.Text(c, "搜索已保存的 Markdown 内容").FontSize(11).TextColor(t.TextMuted)
			if a.searching {
				ui.Text(c, "正在搜索…").TextColor(t.TextMuted)
			} else {
				label := fmt.Sprintf("%d 条匹配", len(a.results.Matches))
				if a.results.Truncated {
					label += "（仅显示前 500 条）"
				}
				ui.Text(c, label).FontSize(12).TextColor(t.TextMuted)
			}
			if a.results.Skipped > 0 {
				ui.Text(c, fmt.Sprintf("已跳过 %d 个无法读取或过大的文件", a.results.Skipped)).FontSize(11).TextColor(t.TextMuted)
			}
			ui.Scroll(c).Grow(1).FillWidth().Gap(6).Children(func() {
				for i, m := range a.results.Matches {
					ui.Button(c.Key(i), "").FillWidth().Padding(10).Children(func() {
						ui.Column(c).Gap(4).Children(func() {
							ui.Text(c, fmt.Sprintf("%s : %d", filepath.Base(m.Path), m.Line)).Bold().FontSize(12)
							ui.Text(c, m.Text).FontSize(12).TextColor(t.TextMuted).SingleLine()
							ui.Text(c, filepath.Dir(m.Path)).FontSize(10).TextColor(t.TextMuted).SingleLine()
						})
					}).OnClick(func() { a.jumpTo(m) })
				}
				if !a.searching && strings.TrimSpace(a.query) != "" && len(a.results.Matches) == 0 {
					ui.Text(c, "没有找到匹配内容").TextColor(t.TextMuted)
				}
			})
			return
		}
		ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() { ui.Text(c, "工作区").FontSize(12).Bold().Grow(1); ui.Button(c, "刷新").OnClick(a.refresh) })
		ui.Scroll(c).Grow(1).FillWidth().Gap(10).Children(func() {
			if len(a.roots) == 0 {
				ui.Text(c, "添加文件夹，开始整理笔记。").TextColor(t.TextMuted)
				ui.Button(c, "选择文件夹").OnClick(a.chooseFolders)
			}
			for _, r := range a.roots {
				ui.Column(c.Key(r.Path)).Gap(4).Children(func() {
					ui.Row(c).Gap(4).Children(func() {
						arrow := "▾ "
						if !r.Open {
							arrow = "▸ "
						}
						ui.Button(c, arrow+r.Name).Grow(1).Tooltip(r.Path).OnClick(func() { r.Open = !r.Open })
						ui.Button(c, "×").Label("移除文件夹 " + r.Name).Tooltip("从工作区移除，不删除文件").OnClick(func() {
							for i, n := range a.roots {
								if n == r {
									a.roots = append(a.roots[:i], a.roots[i+1:]...)
									break
								}
							}
							a.persist()
							if a.searchOpen {
								a.startSearch()
							}
						})
					})
					if r.Open {
						ui.Tree(c, func() {
							for _, n := range r.Children {
								a.treeNode(c, n)
							}
						}).FillWidth()
					}
				})
			}
		})
		ui.Button(c, "＋ 添加文件夹").Disabled(a.busy).FillWidth().OnClick(a.chooseFolders)
	})
}
func (a *app) treeNode(c *ui.Context, n *notebook.Node) {
	if n.Dir {
		ui.TreeItem(c.Key(n.Path), n.Name, &n.Open, func() {
			if !n.Loaded {
				a.fail(notebook.Load(n))
			}
			for _, child := range n.Children {
				a.treeNode(c, child)
			}
		})
	} else {
		selected := false
		if cur := a.current(); cur != nil {
			selected = cur.path == n.Path
		}
		ui.TreeItem(c.Key(n.Path), n.Name, nil, nil).Selected(selected).OnClick(func() { a.openFile(n.Path) })
	}
}
func (a *app) documentView(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).FillHeight().Children(func() {
		if len(a.notes) == 0 {
			ui.Column(c).Fill().Center().Gap(16).Children(func() {
				ui.Text(c, "写下此刻的想法").FontSize(28).Bold()
				ui.Text(c, "纯本地 Markdown 笔记，使用 MyGo Native UI 构建。").TextColor(t.TextMuted)
				ui.Row(c).Gap(10).Children(func() {
					ui.PrimaryButton(c, "创建第一篇笔记").OnClick(a.newNote)
					ui.Button(c, "打开已有笔记").OnClick(a.chooseFile)
				})
				ui.Text(c, "Cmd/Ctrl+N 新建    Cmd/Ctrl+S 保存    Cmd/Ctrl+Shift+F 搜索").FontSize(12).TextColor(t.TextMuted)
			})
			return
		}
		ui.ScrollHorizontal(c).Height(44).FillWidth().Background(t.Surface).Children(func() {
			ui.Row(c).Gap(4).Padding(5, 8).Children(func() {
				for _, n := range a.notes {
					ui.Row(c.Key(n.id)).Gap(2).Children(func() {
						label := n.name()
						if n.dirty() {
							label += " •"
						}
						b := ui.Button(c, label).Tooltip(n.path)
						if n.id == a.active {
							b.Background(t.Background)
						}
						b.OnClick(func() { a.active = n.id; a.persist() })
						ui.Button(c, "×").Label(fmt.Sprintf("关闭标签 %d", n.id)).OnClick(func() { a.guard(n, func() { a.removeNote(n) }) })
					})
				}
			})
		})
		n := a.current()
		if n == nil {
			return
		}
		ui.Row(c).Padding(10, 16).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(t.Border).Children(func() {
			path := n.path
			if path == "" {
				path = "未保存的笔记 · 保存后可粘贴图片"
			}
			ui.Text(c, path).SingleLine().FontSize(12).TextColor(t.TextMuted).Grow(1)
			ui.Segmented(c, &a.mode, "编辑", "预览", "分屏")
			ui.Button(c, "另存为").Disabled(a.busy).OnClick(func() { a.saveAs(n, nil) })
			ui.Button(c, "重新载入").Disabled(n.path == "").OnClick(func() {
				a.guard(n, func() {
					d, e := notebook.Read(n.path)
					if e != nil {
						a.fail(e)
						return
					}
					n.text = d.Text
					n.saved = d.Text
					n.version = d.Version
					n.id = a.nextID + 1
					a.nextID = n.id
					a.active = n.id
					a.configurePreview(n)
				})
			})
		})
		switch a.mode {
		case 0:
			a.editorView(c, n)
		case 1:
			a.previewView(c, n)
		default:
			ui.SplitPercent(c, &a.splitPercent, func() { a.editorView(c, n) }, func() { a.previewView(c, n) }).Grow(1).FillWidth()
		}
	})
}
func (a *app) editorView(c *ui.Context, n *note) {
	in := ui.TextAreaBase(c.Key(fmt.Sprintf("editor-%d", n.id)), &n.text).Label("Markdown 编辑器").Fill().Padding(24).Font("monospace").FontSize(15).OnPaste(func(text string) bool { return a.paste(n, text) })
	if n.insert != "" {
		start, end := in.TextSelection()
		in.ReplaceText(start, end, n.insert)
		n.insert = ""
	}
	if n.jump {
		in.SetTextSelection(n.start, n.end)
		in.Focus()
		n.jump = false
	}
}
func (a *app) previewView(c *ui.Context, n *note) {
	ui.Scroll(c.Key(fmt.Sprintf("preview-%d", n.id))).Fill().Padding(28).TrackScroll(&n.scroll).Children(func() {
		if n.text == "" {
			ui.Text(c, "这里将显示 Markdown 预览").TextColor(c.Theme().TextMuted)
		} else {
			markdown.View(c, n.text, n.md).FillWidth()
		}
	})
}
func (a *app) installMenu() {
	action := func(fn func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) {
			if a.ask || a.busy || a.errorText != "" {
				return
			}
			fn()
			a.win.Invalidate()
		}
	}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
		{Role: mygo.RoleAppMenu},
		{Label: "文件", Submenu: []*mygo.MenuItem{
			{Label: "新建笔记", Accelerator: "CmdOrCtrl+N", Click: action(a.newNote)},
			{Label: "打开文件…", Accelerator: "CmdOrCtrl+O", Click: action(a.chooseFile)},
			{Label: "添加文件夹…", Accelerator: "CmdOrCtrl+Shift+O", Click: action(a.chooseFolders)},
			mygo.Separator(),
			{Label: "保存", Accelerator: "CmdOrCtrl+S", Click: action(func() { a.save(a.current(), nil) })},
			{Label: "另存为…", Accelerator: "CmdOrCtrl+Shift+S", Click: action(func() { a.saveAs(a.current(), nil) })},
			{Label: "关闭笔记", Accelerator: "CmdOrCtrl+W", Click: action(func() {
				if n := a.current(); n != nil {
					a.guard(n, func() { a.removeNote(n) })
				}
			})},
		}},
		{Role: mygo.RoleEditMenu},
		{Label: "视图", Submenu: []*mygo.MenuItem{
			{Label: "显示 / 隐藏侧栏", Accelerator: "CmdOrCtrl+B", Click: action(func() { a.sidebar = !a.sidebar })},
			{Label: "全局搜索", Accelerator: "CmdOrCtrl+Shift+F", Click: action(func() { a.sidebar = true; a.searchOpen = true; a.searchFocus.Focus() })},
			{Label: "编辑 / 预览", Accelerator: "CmdOrCtrl+E", Click: action(func() { a.mode = (a.mode + 1) % 3 })},
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu},
	}))
}
