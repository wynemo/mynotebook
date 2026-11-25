package main

import "github.com/egoist/mygo/ui"

// Paper uses warm neutrals, strong type and a restrained chartreuse accent.
// All controls remain MyGo native widgets, including focus and accessibility.
type paperPalette struct {
	theme                                                                        *ui.Theme
	canvas, sidebar, sheet, line, ink, muted, accent, accentInk, selected, green ui.Color
}

func makePaper(dark bool) paperPalette {
	t := *ui.LightTheme()
	p := paperPalette{canvas: ui.Hex("#f5f4f0"), sidebar: ui.Hex("#eeede7"), sheet: ui.Hex("#fcfbf8"), line: ui.Hex("#deddd5"), ink: ui.Hex("#20211d"), muted: ui.Hex("#85867c"), accent: ui.Hex("#8ca500"), accentInk: ui.Hex("#151a00"), selected: ui.Hex("#e3e8cc"), green: ui.Hex("#329265")}
	if dark {
		t = *ui.DarkTheme()
		p = paperPalette{canvas: ui.Hex("#191a17"), sidebar: ui.Hex("#1f201c"), sheet: ui.Hex("#1c1d19"), line: ui.Hex("#34362e"), ink: ui.Hex("#eeeEE5"), muted: ui.Hex("#909386"), accent: ui.Hex("#d7f64b"), accentInk: ui.Hex("#1b2204"), selected: ui.Hex("#343c20"), green: ui.Hex("#65bd8a")}
	}
	t.Background = p.canvas
	t.Surface = p.sidebar
	t.SurfaceHover = p.selected
	t.SurfacePressed = p.line
	t.Border = p.line
	t.Text = p.ink
	t.TextMuted = p.muted
	t.Accent = p.accent
	t.AccentText = p.accentInk
	t.AccentHover = p.accent
	t.AccentPressed = p.accent
	t.Selection = p.selected
	t.Focus = p.accent
	t.Success = p.green
	t.Radius = 10
	t.FontSize = 13
	t.Spacing = 4
	p.theme = &t
	return p
}

var lightPaper = makePaper(false)
var darkPaper = makePaper(true)

func paper(c *ui.Context) paperPalette {
	if c.Theme().Dark {
		return darkPaper
	}
	return lightPaper
}

// The small SVG vocabulary is drawn in the current text color.
func lineIcon(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.7" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var icons = struct{ book, folder, file, plus, search, sidebar, save, refresh, close, chevron, edit, eye, split, arrow, more, settings *ui.SVG }{
	book:     lineIcon(`<path d="M4 4h13a3 3 0 0 1 3 3v14H7a3 3 0 0 1-3-3V4Z"/><path d="M8 4v17M12 9h5M12 13h4"/>`),
	folder:   lineIcon(`<path d="M3 7V5a1 1 0 0 1 1-1h5l2 3h9a1 1 0 0 1 1 1v11a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V7Z"/>`),
	file:     lineIcon(`<path d="M14 3H5v18h14V8l-5-5Z"/><path d="M14 3v5h5M8 13h8M8 17h5"/>`),
	plus:     lineIcon(`<path d="M12 5v14M5 12h14"/>`),
	search:   lineIcon(`<circle cx="10.5" cy="10.5" r="6.5"/><path d="m16 16 5 5"/>`),
	sidebar:  lineIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M9 4v16"/>`),
	save:     lineIcon(`<path d="M19 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h12l4 4v12a2 2 0 0 1-2 2Z"/><path d="M7 3v6h10V3M7 21v-8h10v8"/>`),
	refresh:  lineIcon(`<path d="M20 7v5h-5M4 17v-5h5"/><path d="M6 7a7 7 0 0 1 12-1l2 3M4 15l2 3a7 7 0 0 0 12-1"/>`),
	close:    lineIcon(`<path d="m7 7 10 10M17 7 7 17"/>`),
	chevron:  lineIcon(`<path d="m9 5 7 7-7 7"/>`),
	edit:     lineIcon(`<path d="m15 4 5 5M4 20l5-1L21 7l-5-5L4 14v6Z"/>`),
	eye:      lineIcon(`<path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12Z"/><circle cx="12" cy="12" r="3"/>`),
	split:    lineIcon(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M12 4v16"/>`),
	arrow:    lineIcon(`<path d="M4 12h16m-6-6 6 6-6 6"/>`),
	more:     lineIcon(`<circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/>`),
	settings: lineIcon(`<path d="M12 3v3M12 18v3M3 12h3M18 12h3M5.6 5.6l2.1 2.1M16.3 16.3l2.1 2.1M5.6 18.4l2.1-2.1M16.3 7.7l2.1-2.1"/><circle cx="12" cy="12" r="5"/><circle cx="12" cy="12" r="1.5"/>`),
}

func paperButton(c *ui.Context, label string, icon *ui.SVG, primary bool) ui.Element {
	p := paper(c)
	b := ui.ButtonBase(c).Label(label).Height(34).Padding(7, 13).Gap(7).Radius(17).FontSize(12).FontWeight(600).TextColor(p.ink)
	if primary {
		b.Background(p.accent).TextColor(p.accentInk)
	} else {
		b.Border(1, p.line)
		if b.Hovered() {
			b.Background(p.selected)
		}
	}
	b.Children(func() {
		if icon != nil {
			ui.Icon(c, icon).Size(15, 15)
		}
		ui.Text(c, label).SingleLine()
	})
	return b
}
func iconButton(c *ui.Context, label string, icon *ui.SVG) ui.Element {
	p := paper(c)
	b := ui.ButtonBase(c).Label(label).Tooltip(label).Size(30, 30).Radius(8).Center().TextColor(p.muted)
	if b.Hovered() {
		b.Background(p.selected).TextColor(p.ink)
	}
	return b.Children(func() { ui.Icon(c, icon).Size(16, 16) })
}
func eyebrow(c *ui.Context, text string) {
	ui.Text(c, text).FontSize(10).FontWeight(650).LetterSpacing(1.3).TextColor(paper(c).muted)
}
func statusDot(c *ui.Context, color ui.Color) {
	ui.Box(c).Size(6, 6).Radius(3).Background(color).Shrink(0)
}
