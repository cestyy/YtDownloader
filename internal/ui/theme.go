package app

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type customTheme struct {
	themeName string
}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch t.themeName {
	case "Pink":
		switch name {
		case theme.ColorNameBackground:
			return color.RGBA{R: 40, G: 42, B: 54, A: 255}
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return color.RGBA{R: 68, G: 71, B: 90, A: 255}
		case theme.ColorNamePrimary:
			return color.RGBA{R: 255, G: 121, B: 198, A: 255}
		case theme.ColorNameForeground:
			return color.RGBA{R: 248, G: 248, B: 242, A: 255}
		}

	case "Ocean":
		switch name {
		case theme.ColorNameBackground:
			return color.RGBA{R: 15, G: 23, B: 42, A: 255}
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return color.RGBA{R: 30, G: 41, B: 59, A: 255}
		case theme.ColorNamePrimary:
			return color.RGBA{R: 56, G: 189, B: 248, A: 255}
		case theme.ColorNameForeground:
			return color.RGBA{R: 241, G: 245, B: 249, A: 255}
		}

	case "Light":
		switch name {
		case theme.ColorNameBackground:
			return color.RGBA{R: 250, G: 250, B: 250, A: 255}
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return color.White
		case theme.ColorNamePrimary:
			return color.RGBA{R: 220, G: 53, B: 69, A: 255}
		case theme.ColorNameForeground:
			return color.RGBA{R: 20, G: 20, B: 20, A: 255}
		}

	default:
		switch name {
		case theme.ColorNameBackground:
			return color.RGBA{R: 15, G: 15, B: 20, A: 255}
		case theme.ColorNameInputBackground, theme.ColorNameButton:
			return color.RGBA{R: 28, G: 32, B: 40, A: 255}
		case theme.ColorNamePrimary:
			return color.RGBA{R: 30, G: 144, B: 255, A: 255}
		case theme.ColorNameForeground:
			return color.RGBA{R: 240, G: 240, B: 240, A: 255}
		}
	}

	v := theme.VariantDark
	if t.themeName == "Light" {
		v = theme.VariantLight
	}
	return theme.DefaultTheme().Color(name, v)
}

func (t *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *customTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
