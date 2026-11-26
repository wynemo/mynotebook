package main

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"mynotebook/internal/notebook"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

func (a *app) openFind() {
	if a.current() == nil {
		return
	}
	a.findOpen = true
	a.findFocus.Focus()
	if a.mode == 1 {
		a.mode = 0
	}
}
func (a *app) editToolbar(c *ui.Context) {
	ui.Row(c).Padding(6, 16).Gap(12).Children(func() {
		ui.Button(c, "查找 / 替换").OnClick(a.openFind)
		ui.Button(c, "插入链接").OnClick(func() { a.beginLink(false) })
		ui.Button(c, "插入图片").OnClick(func() { a.beginLink(true) })
	})
}
func (a *app) findBar(c *ui.Context, n *note) {
	ui.Column(c).Padding(8, 16).Gap(6).Children(func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &a.findText).Label("文内查找").Placeholder("查找文本").Bind(&a.findFocus).Grow(1).OnSubmit(func() { a.findNext(n) })
			ui.Button(c, "下一个").Disabled(a.findText == "").OnClick(func() { a.findNext(n) })
			ui.Button(c, "关闭查找").OnClick(func() { a.findOpen = false })
		})
		ui.Row(c).Gap(8).Children(func() {
			ui.TextInput(c, &a.replaceText).Label("替换为").Placeholder("替换为").Grow(1)
			ui.Button(c, "替换").Disabled(a.findText == "").OnClick(func() { a.replaceOne(n) })
			ui.Button(c, "全部替换").Disabled(a.findText == "").OnClick(func() {
				count := strings.Count(n.text, a.findText)
				if count > 0 {
					a.queueEdit(n, 0, utf8.RuneCountInString(n.text), strings.ReplaceAll(n.text, a.findText, a.replaceText))
				}
				a.status = fmt.Sprintf("已替换 %d 处", count)
			})
		})
	})
}
func (a *app) findNext(n *note) {
	if a.findText == "" {
		return
	}
	runes := []rune(n.text)
	end := min(max(n.end, 0), len(runes))
	tail := string(runes[end:])
	pos := strings.Index(tail, a.findText)
	if pos >= 0 {
		n.start = end + utf8.RuneCountInString(tail[:pos])
	} else {
		pos = strings.Index(n.text, a.findText)
		if pos < 0 {
			a.status = "未找到匹配文本"
			return
		}
		n.start = utf8.RuneCountInString(n.text[:pos])
	}
	n.end = n.start + utf8.RuneCountInString(a.findText)
	n.jump = true
	if a.mode == 1 {
		a.mode = 0
	}
}
func (a *app) replaceOne(n *note) {
	if a.findText == "" {
		return
	}
	r := []rune(n.text)
	start, end := min(n.start, n.end), max(n.start, n.end)
	if start < 0 || end > len(r) || string(r[start:end]) != a.findText {
		a.findNext(n)
		return
	}
	a.queueEdit(n, start, end, a.replaceText)
}
func (a *app) queueEdit(n *note, start, end int, text string) {
	n.editStart, n.editEnd, n.editText, n.editPending = start, end, text, true
	n.start, n.end = start, start+utf8.RuneCountInString(text)
	n.jump = true
	if a.mode == 1 {
		a.mode = 0
	}
}
func (a *app) beginLink(image bool) {
	n := a.current()
	if n == nil {
		return
	}
	a.linkNote = n
	a.imageLink = image
	a.linkOpen = true
	a.linkURL = ""
	a.linkLabel = ""
	r := []rune(n.text)
	start, end := min(n.start, n.end), max(n.start, n.end)
	if start >= 0 && end <= len(r) {
		a.linkLabel = string(r[start:end])
	}
}
func (a *app) insertLink() {
	n := a.linkNote
	if n == nil || strings.TrimSpace(a.linkURL) == "" {
		return
	}
	dest := strings.TrimSpace(a.linkURL)
	if filepath.IsAbs(dest) && n.path != "" {
		if rel, err := filepath.Rel(filepath.Dir(n.path), dest); err == nil {
			dest = filepath.ToSlash(rel)
		}
	}
	// Encode path spaces and parentheses without changing URL delimiters.
	dest = strings.NewReplacer(" ", "%20", "(", "%28", ")", "%29", "\n", "%0A", "\r", "%0D", "<", "%3C", ">", "%3E").Replace(dest)
	label := strings.NewReplacer("\\", "\\\\", "[", "\\[", "]", "\\]").Replace(a.linkLabel)
	prefix := ""
	if a.imageLink {
		prefix = "!"
	}
	a.queueEdit(n, min(n.start, n.end), max(n.start, n.end), prefix+"["+label+"]("+dest+")")
	a.linkOpen = false
}

func (a *app) operationDialogs(c *ui.Context) {
	if a.renameOpen {
		a.smallDialog(c, &a.renameOpen, "重命名", func() {
			ui.TextInput(c, &a.renameName).Label("新名称").Bind(&a.renameFocus).FillWidth().OnSubmit(a.renameEntry)
			if a.renameError != "" {
				ui.Text(c, a.renameError).FontSize(12)
			}
			ui.Button(c, "确定重命名").OnClick(a.renameEntry)
		})
	}
	if a.linkOpen {
		title := "插入链接"
		if a.imageLink {
			title = "插入图片"
		}
		a.smallDialog(c, &a.linkOpen, title, func() {
			ui.TextInput(c, &a.linkLabel).Label("显示文字").Placeholder("显示文字 / 图片说明").FillWidth()
			ui.TextInput(c, &a.linkURL).Label("链接地址").Placeholder("网址或文件路径").FillWidth().OnSubmit(a.insertLink)
			if a.imageLink {
				ui.Button(c, "选择本地图片…").Disabled(a.busy).OnClick(a.chooseImage)
				ui.Text(c, "可输入图片网址或本地路径；也可直接在编辑器粘贴图片。").FontSize(12)
			}
			ui.Button(c, "插入").Disabled(strings.TrimSpace(a.linkURL) == "").OnClick(a.insertLink)
		})
	}
}
func (a *app) smallDialog(c *ui.Context, open *bool, title string, content func()) {
	p := paper(c)
	ui.DialogBase(c, open, func(back, panel ui.Element) {
		back.Background(ui.RGBA(0, 0, 0, 0.35))
		panel.Width(460).Padding(24).Radius(16).Background(p.canvas).Children(func() {
			ui.Column(c).Gap(12).Children(func() {
				ui.Text(c, title).FontSize(18).FontWeight(700)
				content()
				ui.Button(c, "取消").OnClick(func() { *open = false })
			})
		})
	})
}
func (a *app) renameEntry() {
	name := strings.TrimSpace(a.renameName)
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, ".") || strings.ContainsAny(name, "/\\") {
		a.renameError = "请输入有效名称，不能以点开头或包含路径分隔符"
		return
	}
	if notebook.IsMarkdown(a.renamePath) && !notebook.IsMarkdown(name) {
		name += ".md"
	}
	if err := a.movePath(a.renamePath, filepath.Join(filepath.Dir(a.renamePath), name)); err != nil {
		a.renameError = err.Error()
		return
	}
	a.renameOpen = false
}

func (a *app) chooseImage() {
	if a.win == nil || a.busy {
		return
	}
	a.busy = true
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: "选择图片", Filters: []mygo.FileFilter{{Name: "图片", Extensions: []string{"png", "jpg", "jpeg", "gif", "webp", "svg"}}}})
		a.win.Update(func() {
			a.busy = false
			a.fail(err)
			if a.linkOpen && len(paths) > 0 {
				a.linkURL = paths[0]
			}
		})
	}()
}
