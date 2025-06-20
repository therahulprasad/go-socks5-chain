//go:build gui
// +build gui

package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type macOSTheme struct{}

func (m *macOSTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		if variant == theme.VariantDark {
			return color.RGBA{0x1e, 0x1e, 0x1e, 0xff} // Dark gray background
		}
		return color.RGBA{0xf5, 0xf5, 0xf7, 0xff} // Light gray background (macOS light)
	case theme.ColorNameButton:
		if variant == theme.VariantDark {
			return color.RGBA{0x30, 0x30, 0x30, 0xff} // Dark button
		}
		return color.RGBA{0xff, 0xff, 0xff, 0xff} // White button
	case theme.ColorNameForeground:
		if variant == theme.VariantDark {
			return color.RGBA{0xff, 0xff, 0xff, 0xff} // White text on dark
		}
		return color.RGBA{0x1d, 0x1d, 0x1f, 0xff} // Dark text on light
	case theme.ColorNamePrimary:
		return color.RGBA{0x00, 0x7a, 0xff, 0xff} // macOS blue
	case theme.ColorNameSuccess:
		return color.RGBA{0x34, 0xc7, 0x59, 0xff} // macOS green
	case theme.ColorNameWarning:
		return color.RGBA{0xff, 0x95, 0x00, 0xff} // macOS orange
	case theme.ColorNameError:
		return color.RGBA{0xff, 0x45, 0x58, 0xff} // macOS red
	case theme.ColorNameInputBackground:
		if variant == theme.VariantDark {
			return color.RGBA{0x2a, 0x2a, 0x2a, 0xff} // Dark input background
		}
		return color.RGBA{0xff, 0xff, 0xff, 0xff} // White input background
	case theme.ColorNameInputBorder:
		if variant == theme.VariantDark {
			return color.RGBA{0x48, 0x48, 0x48, 0xff} // Dark border
		}
		return color.RGBA{0xd1, 0xd1, 0xd6, 0xff} // Light border
	case theme.ColorNamePlaceHolder:
		if variant == theme.VariantDark {
			return color.RGBA{0x8e, 0x8e, 0x93, 0xff} // Dark placeholder
		}
		return color.RGBA{0x8e, 0x8e, 0x93, 0xff} // Light placeholder
	case theme.ColorNameSeparator:
		if variant == theme.VariantDark {
			return color.RGBA{0x38, 0x38, 0x38, 0xff} // Dark separator
		}
		return color.RGBA{0xe5, 0xe5, 0xe7, 0xff} // Light separator
	case theme.ColorNameShadow:
		if variant == theme.VariantDark {
			return color.RGBA{0x00, 0x00, 0x00, 0x66} // Dark shadow
		}
		return color.RGBA{0x00, 0x00, 0x00, 0x1a} // Light shadow
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (m *macOSTheme) Font(style fyne.TextStyle) fyne.Resource {
	// Use system fonts when possible - Fyne will fall back to defaults
	return theme.DefaultTheme().Font(style)
}

func (m *macOSTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m *macOSTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 13 // macOS standard text size
	case theme.SizeNameHeadingText:
		return 17 // macOS heading size
	case theme.SizeNameSubHeadingText:
		return 15 // macOS subheading size
	case theme.SizeNameCaptionText:
		return 11 // macOS caption size
	case theme.SizeNamePadding:
		return 8 // Tighter padding for modern look
	case theme.SizeNameInnerPadding:
		return 6 // Inner padding
	case theme.SizeNameInputBorder:
		return 1 // Thin borders
	case theme.SizeNameScrollBar:
		return 8 // Thinner scrollbars
	case theme.SizeNameScrollBarSmall:
		return 4 // Very thin scrollbars
	case theme.SizeNameSeparatorThickness:
		return 1 // Thin separators
	}
	return theme.DefaultTheme().Size(name)
}