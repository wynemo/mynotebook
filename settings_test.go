package main

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
)

func TestSettingsApplyAndSurviveRestart(t *testing.T) {
	config := filepath.Join(t.TempDir(), "workspace.json")
	a := newApp(config)
	tt := ui.NewTester(a.view, 1180, 780)
	if e := tt.Click("设置"); e != nil {
		t.Fatal(e)
	}
	if !a.settingsOpen {
		t.Fatal("settings did not open")
	}
	if e := tt.Click("跟随系统"); e != nil {
		t.Fatal(e)
	}
	if e := tt.Click("深色"); e != nil {
		t.Fatal(e)
	}
	if e := tt.Click("切换自动换行"); e != nil {
		t.Fatal(e)
	}
	if a.preferences.Wrap || a.preferences.Theme != "dark" {
		t.Fatalf("preferences not applied: %+v", a.preferences)
	}
	restored := newApp(config)
	if restored.preferences != a.preferences {
		t.Fatalf("preferences not restored: %+v", restored.preferences)
	}
	if e := tt.Click("返回笔记"); e != nil {
		t.Fatal(e)
	}
	if a.settingsOpen {
		t.Fatal("did not return to notes")
	}
}
func TestUpdateStateRejectsStaleResultsAndAllowsRetry(t *testing.T) {
	a := testApp()
	a.update.generation = 2
	a.update.phase = "checking"
	offer := &releaseOffer{version: "0.2.0", notes: "Release notes"}
	a.finishUpdateCheck(1, offer, nil)
	if a.update.offer != nil {
		t.Fatal("stale result accepted")
	}
	a.finishUpdateCheck(2, offer, nil)
	if a.update.phase != "available" || a.update.offer != offer {
		t.Fatal("new update not offered")
	}
	a.update.phase = "installing"
	a.finishUpdateInstall(2, errors.New("signature mismatch"))
	if a.update.phase != "available" {
		t.Fatal("install failure did not allow retry")
	}
	a.finishUpdateInstall(2, nil)
	if a.update.phase != "ready" {
		t.Fatal("successful install not ready")
	}
	a.cancelUpdate()
	a.finishUpdateInstall(2, nil)
	if a.update.phase != "idle" {
		t.Fatal("canceled update accepted a late result")
	}
}
func TestUpdateRestartProtectsDrafts(t *testing.T) {
	original := releaseBackend
	defer func() { releaseBackend = original }()
	restarts := 0
	releaseBackend.relaunch = func() { restarts++ }
	a := testApp()
	a.newNote()
	n := a.current()
	n.text = "未保存的文字"
	a.update.phase = "ready"
	a.restartForUpdate()
	if !a.ask || restarts != 0 {
		t.Fatal("restart bypassed unsaved-note guard")
	}
	tt := ui.NewTester(a.view, 1180, 780)
	if e := tt.Click("取消"); e != nil {
		t.Fatal(e)
	}
	if len(a.notes) != 1 || n.text != "未保存的文字" || restarts != 0 {
		t.Fatal("cancel did not preserve the draft")
	}
	a.restartForUpdate()
	tt.Frame()
	if e := tt.Click("不保存"); e != nil {
		t.Fatal(e)
	}
	if restarts != 1 {
		t.Fatal("confirmed restart did not relaunch")
	}
}
