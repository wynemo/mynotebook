package main

import (
	"fmt"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"mynotebook/internal/notebook"
)

func (a *app) ensurePreferences() {
	if !a.preferencesLoaded {
		a.preferences = notebook.DefaultPreferences()
		a.preferencesLoaded = true
	}
}
func (a *app) loadPreferences() {
	a.ensurePreferences()
	if a.config == "" {
		return
	}
	p, e := notebook.LoadPreferences(filepath.Join(filepath.Dir(a.config), "settings.json"))
	a.fail(e)
	a.preferences = p
}
func (a *app) savePreferences() {
	if a.config != "" {
		a.fail(notebook.SavePreferences(filepath.Join(filepath.Dir(a.config), "settings.json"), a.preferences))
	}
}
func (a *app) settingsView(c *ui.Context) {
	ui.Column(c).Fill().Padding(26, 32).Gap(20).Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(6).Children(func() { eyebrow(c, "MAKE IT YOURS"); ui.Text(c, "设置").FontSize(30).FontWeight(750) })
			ui.Box(c).Grow(1)
			paperButton(c, "返回笔记", icons.arrow, false).OnClick(func() { a.settingsOpen = false })
		})
		ui.Scroll(c).Grow(1).FillWidth().Gap(22).Children(func() {
			a.settingSection(c, "外观与编辑", func() {
				a.settingRow(c, "主题", "选择浅色、深色，或跟随系统。", func() {
					theme := map[string]string{"system": "跟随系统", "light": "浅色", "dark": "深色"}[a.preferences.Theme]
					if ui.Select(c, &theme, []string{"跟随系统", "浅色", "深色"}).Label("主题").Width(145).Changed() {
						a.preferences.Theme = map[string]string{"跟随系统": "system", "浅色": "light", "深色": "dark"}[theme]
						a.savePreferences()
					}
				})
				a.settingRow(c, "编辑字号", "调整 Markdown 源文档的文字大小。", func() {
					size := fmt.Sprintf("%d", int(a.preferences.FontSize))
					if ui.Select(c, &size, []string{"12", "13", "14", "15", "16", "18", "20", "22", "24"}).Label("编辑字号").Width(100).Changed() {
						var n int
						fmt.Sscanf(size, "%d", &n)
						a.preferences.FontSize = float32(n)
						a.savePreferences()
					}
				})
				a.settingRow(c, "自动换行", "长行在编辑区内折行显示。", func() {
					if ui.Switch(c, &a.preferences.Wrap).Label("切换自动换行").Changed() {
						a.savePreferences()
					}
				})
				a.settingRow(c, "默认阅读模式", "新建或打开笔记时采用的模式。", func() {
					mode := a.preferences.DefaultMode
					if ui.Segmented(c, &mode, "编辑", "预览", "分屏").Label("默认阅读模式").Changed() {
						a.preferences.DefaultMode = mode
						a.savePreferences()
					}
				})
			})
			a.settingSection(c, "软件更新", func() {
				a.settingRow(c, "自动检查更新", "启动后检查 GitHub Releases，此后每天检查一次。", func() {
					if ui.Switch(c, &a.preferences.AutomaticUpdates).Label("切换自动检查更新").Changed() {
						a.savePreferences()
						if a.preferences.AutomaticUpdates {
							a.checkUpdates(true)
						} else if a.update.phase == "checking" && a.update.automatic {
							a.cancelUpdate()
						}
					}
				})
				a.settingRow(c, "MyNotebook "+a.version(), "稳定版更新来自 wynemo/mynotebook。下载后验证签名，再由你决定重启。", func() {
					paperButton(c, "检查更新", icons.refresh, false).Disabled(a.update.phase == "checking" || a.update.phase == "installing" || a.update.phase == "ready").OnClick(func() { a.checkUpdates(false) })
				})
				a.updateView(c)
			})
			a.settingSection(c, "关于", func() {
				a.settingRow(c, "你的笔记，只属于你。", "文件与图片保存在本地文件夹。应用更新不会迁移或上传笔记。", func() {
					paperButton(c, "GitHub", icons.arrow, false).OnClick(func() { a.fail(mygo.Shell.OpenExternal("https://github.com/wynemo/mynotebook")) })
				})
			})
		})
	})
}
func (a *app) settingSection(c *ui.Context, title string, content func()) {
	p := paper(c)
	ui.Column(c).FillWidth().Gap(10).Children(func() {
		eyebrow(c, title)
		ui.Column(c).FillWidth().Radius(12).Border(1, p.line).Background(p.sheet).Children(content)
	})
}
func (a *app) settingRow(c *ui.Context, title, detail string, control func()) {
	p := paper(c)
	ui.Row(c).FillWidth().Padding(18, 20).Gap(18).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(p.line).Children(func() {
		ui.Column(c).Grow(1).Gap(6).Children(func() {
			ui.Text(c, title).FontSize(13).FontWeight(650)
			ui.Text(c, detail).FontSize(11).TextColor(p.muted).LineHeight(1.5)
		})
		control()
	})
}
func (a *app) version() string {
	v := mygo.App.Version()
	if v == "" {
		v = "开发版"
	}
	return v
}
