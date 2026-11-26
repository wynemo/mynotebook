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
	a.ensurePreferences()
	p := paper(c)
	if a.preferences.Theme == "light" {
		p = lightPaper
	} else if a.preferences.Theme == "dark" {
		p = darkPaper
	}
	c.SetTheme(p.theme)
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
		a.openSearch()
	}
	if c.Shortcut(mod, ui.KeyF) {
		a.openFind()
	}
	if c.Shortcut(mod, ui.KeyH) {
		a.openFind()
	}
	if c.Shortcut(mod|ui.Shift, ui.KeyT) {
		a.reopenNote()
	}
	if c.Shortcut(mod, ui.KeyK) {
		a.beginLink(false)
	}
	ui.Column(c).Fill().Background(p.canvas).Children(func() {
		a.topbar(c)
		if a.settingsOpen {
			a.settingsView(c)
		} else if a.sidebar {
			ui.Split(c, &a.sidebarWidth, func() { a.sidebarView(c) }, func() { a.documentView(c) }).Grow(1).FillWidth().SplitReset(270).OnChange(a.persist)
		} else {
			a.documentView(c)
		}
		a.statusbar(c)
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
	a.createDialog(c)
	a.operationDialogs(c)
	if a.errorText != "" {
		show := true
		if ui.AlertDialog(c, &show, "操作未完成", a.errorText, "知道了") >= 0 || !show {
			a.errorText = ""
		}
	}
}
func (a *app) openSearch() {
	a.sidebar = true
	a.searchOpen = true
	a.searchFocus.Focus()
	a.startSearch()
}
func (a *app) topbar(c *ui.Context) {
	p := paper(c)
	ui.Row(c).Height(76).Padding(18, 24).Gap(14).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.line).Children(func() {
		ui.Box(c).Size(36, 36).Radius(10).Background(p.accent).TextColor(p.accentInk).Center().Children(func() { ui.Icon(c, icons.book).Size(23, 23) })
		ui.Column(c).Gap(2).Children(func() {
			ui.Text(c, "MyNotebook").FontSize(19).FontWeight(750).LetterSpacing(-0.5)
			eyebrow(c, "A LITTLE SPACE FOR YOUR MIND")
		})
		ui.Box(c).Grow(1)
		iconButton(c, "设置", icons.settings).OnClick(func() { a.settingsOpen = !a.settingsOpen })
		iconButton(c, "侧栏", icons.sidebar).OnClick(func() { a.sidebar = !a.sidebar })
		iconButton(c, "全局搜索", icons.search).OnClick(a.openSearch)
		ui.Box(c).Size(1, 22).Background(p.line)
		paperButton(c, "打开文件", icons.folder, false).Disabled(a.busy).OnClick(a.chooseFile)
		paperButton(c, "新建笔记", icons.plus, true).OnClick(a.newNote)
	})
}
func (a *app) statusbar(c *ui.Context) {
	p := paper(c)
	ui.Row(c).Height(34).Padding(8, 24).Gap(8).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(p.line).Children(func() {
		statusDot(c, p.green)
		ui.Text(c, "本地笔记").FontSize(11).TextColor(p.muted)
		ui.Text(c, "/ "+a.status).FontSize(11).TextColor(p.muted).SingleLine().Grow(1)
		if n := a.current(); n != nil {
			state := "已保存"
			if n.dirty() {
				state = "未保存"
			}
			ui.Text(c, fmt.Sprintf("%s    ·    %d 字符    ·    UTF-8", state, utf8.RuneCountInString(n.text))).FontSize(11).TextColor(p.muted)
		} else {
			ui.Text(c, fmt.Sprintf("%d 个工作区", len(a.roots))).FontSize(11).TextColor(p.muted)
		}
	})
}
func (a *app) sidebarView(c *ui.Context) {
	p := paper(c)
	ui.Column(c).Fill().Background(p.sidebar).Padding(20, 16).Gap(18).Children(func() {
		ui.Row(c).Gap(4).Padding(4).Radius(12).Background(p.canvas).Children(func() {
			a.navItem(c, "文件夹", icons.folder, !a.searchOpen, func() { a.searchOpen = false })
			a.navItem(c, "搜索", icons.search, a.searchOpen, a.openSearch)
		})
		if a.searchOpen {
			a.searchView(c)
			return
		}
		ui.Row(c).Gap(5).AlignItems(ui.Center).Children(func() {
			eyebrow(c, "工作区 / WORKSPACES")
			ui.Box(c).Grow(1)
			iconButton(c, "刷新", icons.refresh).OnClick(a.refresh)
		})
		ui.Scroll(c).Grow(1).FillWidth().Gap(18).ContextMenu(func(m *ui.Menu) {
			if len(a.roots) == 1 {
				a.creationMenu(m, a.roots[0].Path)
			} else {
				for _, r := range a.roots {
					m.Submenu(r.Name, func(sub *ui.Menu) { a.creationMenu(sub, r.Path) })
				}
			}
		}).Children(func() {
			if len(a.roots) == 0 {
				ui.Column(c).Padding(12, 6).Gap(10).Children(func() {
					ui.Text(c, "让想法各就各位。").FontWeight(650)
					ui.Text(c, "添加一个文件夹，把散落的 Markdown 笔记放在手边。").FontSize(12).LineHeight(1.6).TextColor(p.muted)
				})
			}
			for _, r := range a.roots {
				ui.Column(c.Key(r.Path)).Gap(7).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(3).Children(func() {
						b := ui.ButtonBase(c).Label(r.Name).Grow(1).Justify(ui.Start).Padding(6, 4).Gap(7).Tooltip(r.Path).Children(func() {
							arrow := ui.Icon(c, icons.chevron).Size(11, 11).TextColor(p.muted)
							if r.Open {
								arrow.Rotate(90)
							}
							ui.Icon(c, icons.folder).Size(16, 16).TextColor(p.muted)
							ui.Text(c, r.Name).FontSize(12).FontWeight(700).SingleLine()
						})
						b.OnClick(func() { r.Open = !r.Open })
						a.fileRow(c, b, r)
						iconButton(c, "移除文件夹 "+r.Name, icons.close).OnClick(func() { a.removeRoot(r) })
					})
					if r.Open {
						ui.Tree(c, func() {
							for _, n := range r.Children {
								a.treeNode(c, n)
							}
						}).FillWidth().FontSize(13)
					}
				})
			}
		})
		ui.Column(c).Gap(12).Children(func() {
			paperButton(c, "添加文件夹", icons.plus, false).FillWidth().Disabled(a.busy).OnClick(a.chooseFolders)
			ui.Row(c).Gap(6).AlignItems(ui.Center).Padding(0, 4).Children(func() {
				statusDot(c, p.green)
				ui.Text(c, "文件始终保存在你的电脑上").FontSize(10).TextColor(p.muted)
			})
		})
	})
}
func (a *app) removeRoot(r *notebook.Node) {
	for i, n := range a.roots {
		if n == r {
			a.roots = append(a.roots[:i], a.roots[i+1:]...)
			break
		}
	}
	a.persist()
	a.startSearch()
	a.syncWatchers()
}
func (a *app) navItem(c *ui.Context, label string, icon *ui.SVG, selected bool, action func()) {
	p := paper(c)
	b := ui.ButtonBase(c).Label(label).Grow(1).Height(32).Gap(7).Padding(8, 12).Radius(9).FontSize(12).FontWeight(600).TextColor(p.muted)
	if selected {
		b.Background(p.accent).TextColor(p.accentInk)
	} else if b.Hovered() {
		b.Background(p.selected).TextColor(p.ink)
	}
	b.Children(func() { ui.Icon(c, icon).Size(15, 15); ui.Text(c, label) }).OnClick(action)
}
func (a *app) searchView(c *ui.Context) {
	p := paper(c)
	search := ui.SearchField(c, &a.query).Label("全局搜索").Placeholder("寻找一个想法…").Bind(&a.searchFocus).Height(38).Radius(10).Background(p.sheet)
	if search.Changed() {
		a.startSearch()
	}
	label := fmt.Sprintf("%d 条匹配", len(a.results.Matches))
	if a.searching {
		label = "正在搜索…"
	} else if a.results.Truncated {
		label = "前 500 条匹配"
	}
	ui.Row(c).AlignItems(ui.Center).Children(func() {
		eyebrow(c, "搜索结果")
		ui.Box(c).Grow(1)
		ui.Text(c, label).FontSize(11).TextColor(p.muted)
	})
	ui.Scroll(c).Grow(1).FillWidth().Gap(8).Children(func() {
		for i, m := range a.results.Matches {
			b := ui.ButtonBase(c.Key(i)).Label(fmt.Sprintf("%s : %d", filepath.Base(m.Path), m.Line)).FillWidth().Justify(ui.Start).Padding(12).Radius(10).Background(p.canvas).Border(1, p.line)
			if b.Hovered() {
				b.Background(p.selected)
			}
			b.Children(func() {
				ui.Column(c).Gap(7).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, icons.file).Size(14, 14).TextColor(p.muted)
						ui.Text(c, filepath.Base(m.Path)).Bold().FontSize(12).SingleLine().Grow(1)
						ui.Text(c, fmt.Sprintf("L%d", m.Line)).FontSize(10).TextColor(p.muted)
					})
					ui.Text(c, m.Text).FontSize(12).LineHeight(1.5).TextColor(p.muted).MaxLines(2)
				})
			}).OnClick(func() { a.jumpTo(m) })
		}
		if !a.searching && len(a.results.Matches) == 0 {
			ui.Column(c).Padding(12, 4).Gap(8).Children(func() {
				if strings.TrimSpace(a.query) == "" {
					ui.Text(c, "想找什么？").FontSize(18).Bold()
					ui.Text(c, "输入关键词，搜索所有工作区里的笔记。").TextColor(p.muted).FontSize(12)
				} else {
					ui.Text(c, "没有找到匹配内容").TextColor(p.muted)
				}
			})
		}
	})
	detail := "搜索已保存的 Markdown 内容"
	if a.results.Skipped > 0 {
		detail = fmt.Sprintf("跳过 %d 个无法读取或过大的文件", a.results.Skipped)
	}
	ui.Text(c, detail).FontSize(10).TextColor(p.muted)
}
func (a *app) treeNode(c *ui.Context, n *notebook.Node) {
	if n.Dir {
		row := ui.TreeItem(c.Key(n.Path), n.Name, &n.Open, func() {
			if !n.Loaded {
				a.fail(notebook.Load(n))
			}
			for _, child := range n.Children {
				a.treeNode(c, child)
			}
		})
		a.fileRow(c, row, n)
	} else {
		selected := false
		if cur := a.current(); cur != nil {
			selected = cur.path == n.Path
		}
		row := ui.TreeItem(c.Key(n.Path), n.Name, nil, nil).Selected(selected).OnClick(func() { a.openFile(n.Path) })
		a.fileRow(c, row, n)
	}
}
func (a *app) documentView(c *ui.Context) {
	p := paper(c)
	ui.Column(c).Grow(1).FillHeight().Background(p.canvas).Children(func() {
		if len(a.notes) == 0 {
			a.emptyView(c)
			return
		}
		ui.ScrollHorizontal(c).Height(50).FillWidth().BorderWidth(0, 0, 1, 0).BorderColor(p.line).Children(func() {
			ui.Row(c).Gap(6).Padding(10, 22).Children(func() {
				for _, n := range a.notes {
					ui.Row(c.Key(n.id)).Gap(1).AlignItems(ui.Center).Radius(8).Children(func() {
						b := ui.ButtonBase(c).Label(n.name()).Tooltip(n.path).Height(30).Padding(6, 10).Gap(7).Radius(8).TextColor(p.muted)
						if n.id == a.active {
							b.Background(p.selected).TextColor(p.ink)
						} else if b.Hovered() {
							b.Background(p.sidebar)
						}
						b.Children(func() {
							ui.Icon(c, icons.file).Size(14, 14)
							ui.Text(c, n.name()).FontSize(12).FontWeight(550)
							if n.dirty() {
								statusDot(c, p.accent)
							}
						}).OnClick(func() { a.active = n.id; a.persist() }).ContextMenu(func(m *ui.Menu) { a.tabMenu(m, n) })
						iconButton(c, fmt.Sprintf("关闭标签 %d", n.id), icons.close).Size(22, 26).OnClick(func() { a.guard(n, func() { a.removeNote(n) }) })
					})
				}
			})
		})
		n := a.current()
		if n == nil {
			return
		}
		if n.external != "" {
			ui.Row(c).Padding(8, 28).Gap(12).Children(func() {
				ui.Text(c, n.external).Grow(1).FontSize(12)
				ui.Button(c, "另存为…").OnClick(func() { a.saveAs(n, nil) })
				ui.Button(c, "重新载入").OnClick(func() { a.reload(n) })
			})
		}
		ui.Row(c).Padding(22, 28).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Grow(1).Gap(7).Children(func() {
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, icons.folder).Size(12, 12).TextColor(p.muted)
					ui.Text(c, a.breadcrumb(n)).SingleLine().FontSize(11).TextColor(p.muted)
				})
				title := strings.TrimSuffix(n.name(), filepath.Ext(n.name()))
				if n.path == "" {
					title = "未命名笔记"
				}
				ui.Text(c, title).FontSize(26).FontWeight(750).LetterSpacing(-0.6).SingleLine()
			})
			paperButton(c, "保存", icons.save, false).Disabled(a.busy).OnClick(func() { a.save(n, nil) })
			iconButton(c, "更多操作", icons.more).Menu(func(m *ui.Menu) {
				if m.Item("另存为…").Disabled(a.busy).Chosen() {
					a.saveAs(n, nil)
				}
				if m.Item("重新载入").Disabled(n.path == "").Chosen() {
					a.reload(n)
				}
			})
		})
		ui.Column(c).Grow(1).FillWidth().Padding(0, 28, 22, 28).Children(func() {
			ui.Column(c).Fill().Radius(12).Border(1, p.line).Background(p.sheet).Children(func() {
				ui.Row(c).Height(48).Padding(9, 16).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.line).Children(func() {
					eyebrow(c, "MARKDOWN")
					ui.Box(c).Grow(1)
					for i, label := range []string{"编辑", "预览", "分屏"} {
						icon := []*ui.SVG{icons.edit, icons.eye, icons.split}[i]
						b := ui.ButtonBase(c).Label(label).Height(28).Padding(6, 10).Gap(5).Radius(7).TextColor(p.muted)
						if a.mode == i {
							b.Background(p.selected).TextColor(p.ink)
						} else if b.Hovered() {
							b.Background(p.sidebar)
						}
						b.Children(func() { ui.Icon(c, icon).Size(13, 13); ui.Text(c, label).FontSize(11).FontWeight(600) }).OnClick(func() { a.mode = i })
					}
				})
				a.editToolbar(c)
				if a.findOpen {
					a.findBar(c, n)
				}
				switch a.mode {
				case 0:
					a.editorView(c, n)
				case 1:
					a.previewView(c, n)
				default:
					ui.SplitPercent(c, &a.splitPercent, func() { a.editorView(c, n) }, func() { a.previewView(c, n) }).Grow(1).FillWidth()
				}
			})
		})
	})
}
func (a *app) breadcrumb(n *note) string {
	if n.path == "" {
		return "草稿 / 保存后可粘贴图片"
	}
	for _, r := range a.roots {
		if rel, e := filepath.Rel(r.Path, n.path); e == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return r.Name + " / " + filepath.ToSlash(rel)
		}
	}
	return filepath.Base(filepath.Dir(n.path)) + " / " + n.name()
}
func (a *app) reload(n *note) {
	a.guard(n, func() {
		d, e := notebook.Read(n.path)
		if e != nil {
			a.fail(e)
			return
		}
		n.external = ""
		n.text = d.Text
		n.saved = d.Text
		n.version = d.Version
		a.nextID++
		n.id = a.nextID
		a.active = n.id
		a.configurePreview(n)
	})
}
func (a *app) editorView(c *ui.Context, n *note) {
	p := paper(c)
	in := ui.TextAreaBase(c.Key(fmt.Sprintf("editor-%d", n.id)), &n.text).Label("Markdown 编辑器").Placeholder("从一句话开始…").Fill().Padding(26).Font("monospace").FontSize(a.preferences.FontSize).LineHeight(1.7).TextColor(p.ink).OnPaste(func(text string) bool { return a.paste(n, text) })
	if !a.preferences.Wrap {
		in.NoWrap()
	}
	if n.editPending {
		in.ReplaceText(n.editStart, n.editEnd, n.editText)
		n.editPending = false
	}
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
	n.start, n.end = in.TextSelection()
}

var lightMarkdown = &markdown.Theme{FontSize: 15, LineHeight: 1.75, BlockSpacing: 1.1, Link: ui.Hex("#728700"), QuoteBar: ui.Hex("#8ca500"), CodeBlockRadius: 8, CodeBlockBackground: ui.Hex("#f1f0e9")}
var darkMarkdown = &markdown.Theme{FontSize: 15, LineHeight: 1.75, BlockSpacing: 1.1, Link: ui.Hex("#d7f64b"), QuoteBar: ui.Hex("#d7f64b"), CodeBlockRadius: 8, CodeBlockBackground: ui.Hex("#25271f")}

func (a *app) previewView(c *ui.Context, n *note) {
	if c.Theme().Dark {
		n.md.Theme = darkMarkdown
	} else {
		n.md.Theme = lightMarkdown
	}
	ui.Scroll(c.Key(fmt.Sprintf("preview-%d", n.id))).Fill().Padding(28).TrackScroll(&n.scroll).Children(func() {
		if n.text == "" {
			ui.Text(c, "这里将显示 Markdown 预览").TextColor(paper(c).muted)
		} else {
			markdown.View(c, n.text, n.md).FillWidth()
		}
	})
}
func (a *app) emptyView(c *ui.Context) {
	p := paper(c)
	ui.Scroll(c).Fill().Padding(48, 46).Children(func() {
		ui.Column(c).MaxWidth(780).FillWidth().Gap(24).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() { statusDot(c, p.green); eyebrow(c, "YOUR NOTES, YOUR SPACE") })
			ui.Column(c).Gap(8).Children(func() {
				ui.Text(c, "留一点空间，").FontSize(44).FontWeight(800).LetterSpacing(-1.4)
				ui.Text(c, "给新的想法。").FontSize(44).FontWeight(800).LetterSpacing(-1.4).TextColor(p.accent)
			})
			ui.Text(c, "一篇笔记、一段灵感，或是还没想好名字的计划。\n从这里开始，把思绪慢慢写清楚。").FontSize(14).LineHeight(1.8).TextColor(p.muted)
			ui.Row(c).Gap(12).Padding(8, 0, 12, 0).Children(func() {
				paperButton(c, "创建第一篇笔记", icons.plus, true).Height(40).Radius(20).OnClick(a.newNote)
				paperButton(c, "打开已有笔记", icons.arrow, false).Height(40).Radius(20).OnClick(a.chooseFile)
			})
			ui.Box(c).Height(1).FillWidth().Background(p.line)
			ui.Row(c).Gap(0).Children(func() {
				a.welcomeCell(c, "01", "写下想法", "用 Markdown 保持简单，\n让表达成为唯一的重点。")
				a.welcomeCell(c, "02", "放在一起", "添加多个文件夹，\n在熟悉的目录里整理笔记。")
				a.welcomeCell(c, "03", "随时回望", "预览、搜索、图片附件，\n让每个想法都有迹可循。")
			})
			ui.Row(c).Padding(16, 0).Gap(22).Children(func() {
				a.keyHint(c, "新建", "N")
				a.keyHint(c, "保存", "S")
				a.keyHint(c, "全局搜索", "⇧ F")
			})
		})
	})
}
func (a *app) welcomeCell(c *ui.Context, num, title, detail string) {
	p := paper(c)
	ui.Column(c).Grow(1).Gap(12).Padding(10, 14, 10, 0).Children(func() {
		eyebrow(c, num)
		ui.Text(c, title).FontSize(17).FontWeight(700)
		ui.Text(c, detail).FontSize(12).LineHeight(1.7).TextColor(p.muted)
	})
}
func (a *app) keyHint(c *ui.Context, label, key string) {
	p := paper(c)
	mod := "Ctrl "
	if runtime.GOOS == "darwin" {
		mod = "⌘ "
	}
	ui.Row(c).Gap(7).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(p.muted)
		ui.Text(c, mod+key).FontSize(10).Padding(3, 6).Radius(4).Border(1, p.line).TextColor(p.muted)
	})
}
func (a *app) installMenu() {
	action := func(fn func()) func(*mygo.MenuItem, *mygo.Window) {
		return func(*mygo.MenuItem, *mygo.Window) {
			if a.ask || a.busy || a.createOpen || a.renameOpen || a.linkOpen || a.errorText != "" {
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
			{Label: "重新打开关闭的笔记", Accelerator: "CmdOrCtrl+Shift+T", Click: action(a.reopenNote)},
			{Label: "关闭笔记", Accelerator: "CmdOrCtrl+W", Click: action(func() {
				if n := a.current(); n != nil {
					a.guard(n, func() { a.removeNote(n) })
				}
			})},
		}},
		{Label: "编辑", Submenu: []*mygo.MenuItem{
			{Role: mygo.RoleUndo}, {Role: mygo.RoleRedo}, mygo.Separator(),
			{Role: mygo.RoleCut}, {Role: mygo.RoleCopy}, {Role: mygo.RolePaste}, {Role: mygo.RoleSelectAll},
			mygo.Separator(),
			{Label: "文内查找 / 替换", Accelerator: "CmdOrCtrl+F", Click: action(a.openFind)},
			{Label: "插入链接", Accelerator: "CmdOrCtrl+K", Click: action(func() { a.beginLink(false) })},
			{Label: "插入图片", Click: action(func() { a.beginLink(true) })},
		}},
		{Label: "视图", Submenu: []*mygo.MenuItem{
			{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: action(func() { a.settingsOpen = !a.settingsOpen })},
			{Label: "检查更新…", Click: action(func() { a.settingsOpen = true; a.checkUpdates(false) })},
			mygo.Separator(),
			{Label: "显示 / 隐藏侧栏", Accelerator: "CmdOrCtrl+B", Click: action(func() { a.sidebar = !a.sidebar })},
			{Label: "全局搜索", Accelerator: "CmdOrCtrl+Shift+F", Click: action(func() { a.sidebar = true; a.searchOpen = true; a.searchFocus.Focus() })},
			{Label: "编辑 / 预览", Accelerator: "CmdOrCtrl+E", Click: action(func() { a.mode = (a.mode + 1) % 3 })},
			{Role: mygo.RoleToggleFullScreen},
		}},
		{Role: mygo.RoleWindowMenu},
	}))
}

func (a *app) createDialog(c *ui.Context) {
	if !a.createOpen {
		return
	}
	p := paper(c)
	ui.DialogBase(c, &a.createOpen, func(backdrop, panel ui.Element) {
		backdrop.Background(ui.RGBA(0, 0, 0, 0.35))
		panel.Width(420).Padding(24).Radius(16).Background(p.canvas).Children(func() {
			ui.Column(c).Gap(14).Children(func() {
				title := "新建文件"
				if a.createDir {
					title = "新建文件夹"
				}
				ui.Text(c, title).FontSize(18).FontWeight(700)
				ui.Text(c, a.createParent).FontSize(12).TextColor(p.muted)
				ui.TextInput(c, &a.createName).Label("名称").Bind(&a.createFocus).FillWidth().OnSubmit(a.createEntry)
				if a.createError != "" {
					ui.Text(c, a.createError).FontSize(12)
				}
				ui.Row(c).Gap(10).Children(func() {
					paperButton(c, "取消", icons.close, false).OnClick(func() { a.createOpen = false })
					paperButton(c, "创建", icons.plus, true).OnClick(a.createEntry)
				})
			})
		})
	})
}
