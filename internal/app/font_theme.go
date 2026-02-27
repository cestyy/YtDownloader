package app

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed fonts/Roboto-Regular.ttf
var robotoRegular []byte

type embeddedFontTheme struct {
	fyne.Theme
}

func (t embeddedFontTheme) Font(s fyne.TextStyle) fyne.Resource {
	return fyne.NewStaticResource("Roboto-Regular.ttf", robotoRegular)
}

func applyEmbeddedFont(a fyne.App) {
	a.Settings().SetTheme(embeddedFontTheme{Theme: theme.DefaultTheme()})
}
