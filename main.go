package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"mynotebook/internal/notebook"
)

func main() {
	mygo.App.SetName("MyNotebook")
	mygo.App.WhenReady(func() {
		base, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		a := newApp(filepath.Join(base, "MyNotebook", "workspace.json"))
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "MyNotebook", Width: 1180, Height: 780, MinWidth: 760, MinHeight: 500, Content: ui.View(a.view)})
		a.installMenu()
		a.win.OnClose(func(e *mygo.CloseEvent) {
			if a.hasChanges() {
				e.PreventDefault()
				a.requestQuit()
				a.win.Invalidate()
			} else {
				a.persist()
			}
		})
		mygo.App.OnBeforeQuit(func(e *mygo.QuitEvent) {
			if a.hasChanges() {
				e.PreventDefault()
				a.requestQuit()
				a.win.Invalidate()
			}
		})
		for _, path := range os.Args[1:] {
			if notebook.IsMarkdown(path) {
				a.openFile(path)
			}
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
