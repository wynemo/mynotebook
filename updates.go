package main

import (
	"context"
	"fmt"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
)

// MyGo verifies the Ed25519 signature before replacing installed files.
// The app owns the UI so a restart can go through the unsaved-note guard.
type releaseOffer struct {
	version, notes string
	install        func(context.Context, func(int64, int64)) error
}
type updateState struct {
	phase, message string
	offer          *releaseOffer
	progress       float64
	checkedAt      time.Time
	automatic      bool
	generation     int
	cancel         context.CancelFunc
}

var releaseBackend = struct {
	enabled  func() bool
	check    func(context.Context) (*releaseOffer, error)
	relaunch func()
}{
	enabled: mygo.Updater.Enabled,
	check: func(ctx context.Context) (*releaseOffer, error) {
		up, e := mygo.Updater.Check(ctx)
		if e != nil || up == nil {
			return nil, e
		}
		return &releaseOffer{version: up.Version, notes: up.Notes, install: up.Install}, nil
	},
	relaunch: func() { mygo.App.Relaunch() },
}

func (a *app) checkUpdates(automatic bool) {
	if a.update.phase == "checking" || a.update.phase == "installing" || a.update.phase == "ready" {
		return
	}
	if !releaseBackend.enabled() {
		if !automatic {
			a.update.phase = "disabled"
			a.update.message = "当前为开发运行或安装位置不可写。请从 GitHub Releases 安装正式版本后使用应用内升级。"
		}
		return
	}
	if a.win == nil {
		return
	}
	a.update.generation++
	id := a.update.generation
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	a.update.cancel = cancel
	a.update.phase = "checking"
	a.update.message = "正在检查 GitHub Releases…"
	a.update.automatic = automatic
	go func() {
		defer cancel()
		offer, e := releaseBackend.check(ctx)
		a.win.Update(func() { a.finishUpdateCheck(id, offer, e) })
	}()
}
func (a *app) finishUpdateCheck(id int, offer *releaseOffer, e error) {
	if a.update.generation != id {
		return
	}
	a.update.cancel = nil
	if e != nil {
		a.update.phase = "error"
		a.update.message = "检查更新失败，请稍后重试。"
		if !a.update.automatic {
			a.update.message += " " + e.Error()
		}
		return
	}
	a.update.checkedAt = time.Now()
	a.update.offer = offer
	if offer == nil {
		a.update.phase = "current"
		a.update.message = "已是最新版本"
		return
	}
	a.update.phase = "available"
	a.update.message = "发现新版本 " + offer.version
	a.status = "发现新版本 " + offer.version + " · 在设置中更新"
}
func (a *app) installUpdate() {
	if a.update.offer == nil || a.update.offer.install == nil || a.update.phase != "available" || a.win == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	a.update.cancel = cancel
	a.update.phase = "installing"
	a.update.message = "正在下载并验证更新签名…"
	a.update.progress = 0
	a.update.generation++
	id := a.update.generation
	offer := a.update.offer
	go func() {
		defer cancel()
		e := offer.install(ctx, func(done, total int64) {
			a.win.Update(func() {
				if a.update.generation == id && total > 0 {
					a.update.progress = min(1, float64(done)/float64(total))
				}
			})
		})
		a.win.Update(func() { a.finishUpdateInstall(id, e) })
	}()
}
func (a *app) finishUpdateInstall(id int, e error) {
	if a.update.generation != id {
		return
	}
	a.update.cancel = nil
	if e != nil {
		a.update.phase = "available"
		a.update.message = "更新未完成，当前版本保留。 " + e.Error()
		return
	}
	a.update.phase = "ready"
	a.update.progress = 1
	a.update.message = "更新已安装，重启后生效。"
	a.status = "更新已就绪 · 在设置中重启"
}
func (a *app) cancelUpdate() {
	if a.update.cancel != nil {
		a.update.cancel()
		a.update.cancel = nil
	}
	a.update.generation++
	a.update.phase = "idle"
	a.update.message = "已取消"
}
func (a *app) restartForUpdate() {
	if a.update.phase != "ready" {
		return
	}
	if a.ask {
		return
	}
	for _, n := range a.notes {
		if n.dirty() {
			a.settingsOpen = false
			a.active = n.id
			a.guard(n, func() { a.removeNote(n); a.restartForUpdate() })
			return
		}
	}
	a.persist()
	releaseBackend.relaunch()
}
func (a *app) startUpdateChecks() {
	ctx, cancel := context.WithCancel(context.Background())
	a.stopUpdateChecks = cancel
	go func() {
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				a.win.Update(func() {
					if a.preferences.AutomaticUpdates {
						a.checkUpdates(true)
					}
				})
				timer.Reset(24 * time.Hour)
			}
		}
	}()
}
func (a *app) stopUpdates() {
	if a.stopUpdateChecks != nil {
		a.stopUpdateChecks()
	}
	if a.update.cancel != nil {
		a.update.cancel()
	}
}
func (a *app) updateView(c *ui.Context) {
	p := paper(c)
	ui.Column(c).FillWidth().Padding(16, 20).Gap(12).Children(func() {
		if a.update.message != "" {
			ui.Text(c, a.update.message).FontSize(12).TextColor(p.muted)
		}
		if !a.update.checkedAt.IsZero() {
			ui.Text(c, "上次检查："+a.update.checkedAt.Format("2006-01-02 15:04")).FontSize(10).TextColor(p.muted)
		}
		if a.update.phase == "available" && a.update.offer != nil {
			if a.update.offer.notes != "" {
				ui.Scroll(c).Height(130).FillWidth().Children(func() { markdown.View(c, a.update.offer.notes, nil) })
			}
			paperButton(c, "下载并安装 "+a.update.offer.version, icons.arrow, true).OnClick(a.installUpdate)
		}
		if a.update.phase == "installing" {
			ui.Progress(c, a.update.progress).FillWidth()
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, fmt.Sprintf("%.0f%%", a.update.progress*100)).FontSize(11).TextColor(p.muted).Grow(1)
				paperButton(c, "取消下载", icons.close, false).OnClick(a.cancelUpdate)
			})
		}
		if a.update.phase == "checking" {
			paperButton(c, "取消检查", icons.close, false).OnClick(a.cancelUpdate)
		}
		if a.update.phase == "ready" {
			paperButton(c, "重启并完成更新", icons.refresh, true).OnClick(a.restartForUpdate)
		}
	})
}
