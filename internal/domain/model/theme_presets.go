package model

// PresetThemes maps theme names to their full Theme definitions.
var PresetThemes = map[string]Theme{
	"ios-dark":         iosDark,
	"amber-retro":      amberRetro,
	"catppuccin-mocha": catppuccinMocha,
	"nord":             nord,
	"dracula":          dracula,
}

// PresetNames defines the order for cycling through themes.
var PresetNames = []string{
	"ios-dark",
	"amber-retro",
	"catppuccin-mocha",
	"nord",
	"dracula",
}

// DefaultThemeName is the theme loaded when no preference is set.
const DefaultThemeName = "ios-dark"

var iosDark = Theme{
	Name:               "ios-dark",
	HeaderTitle:        "▶ GoYT",
	Background:         "#000000",
	Surface:            "#1C1C1E",
	TextPrimary:        "#FFFFFF",
	TextSecondary:      "#8E8E93",
	TextOnAccent:       "#FFFFFF",
	PrimaryHighlight:   "#0A84FF",
	SecondaryHighlight: "#5E5CE6",
	InactiveBorder:     "#38383A",
	Success:            "#30D158",
	Error:              "#FF453A",
	Warning:            "#FFD60A",
	Info:               "#64D2FF",
	Muted:              "#48484A",
	VisualizerPlayed:   []RGB{{R: 10, G: 132, B: 255}, {R: 94, G: 92, B: 230}, {R: 191, G: 90, B: 242}},
	VisualizerUnplayed: []RGB{{R: 5, G: 40, B: 80}, {R: 30, G: 28, B: 72}, {R: 58, G: 28, B: 75}},
	EqualizerChar:      '●',
	EqualizerBg:        "#1C1C1E",
}

var amberRetro = Theme{
	Name:               "amber-retro",
	HeaderTitle:        "▶ GoYT — DEC VT220 Amber Audio Terminal",
	Background:         "#000000",
	Surface:            "#1A1200",
	TextPrimary:        "#FFB000",
	TextSecondary:      "#997A33",
	TextOnAccent:       "#121212",
	PrimaryHighlight:   "#FFB000",
	SecondaryHighlight: "#FFD700",
	InactiveBorder:     "#3C3C3C",
	Success:            "#FFD700",
	Error:              "#FF6600",
	Warning:            "#FFB000",
	Info:               "#FFCC33",
	Muted:              "#5A4000",
	VisualizerPlayed:   []RGB{{R: 211, G: 84, B: 0}, {R: 254, G: 153, B: 0}, {R: 255, G: 215, B: 0}},
	VisualizerUnplayed: []RGB{{R: 62, G: 35, B: 0}, {R: 90, G: 57, B: 0}, {R: 122, G: 82, B: 0}},
	EqualizerChar:      '●',
	EqualizerBg:        "#1C1C1C",
}

var catppuccinMocha = Theme{
	Name:               "catppuccin-mocha",
	HeaderTitle:        "▶ GoYT",
	Background:         "#1E1E2E",
	Surface:            "#313244",
	TextPrimary:        "#CDD6F4",
	TextSecondary:      "#A6ADC8",
	TextOnAccent:       "#1E1E2E",
	PrimaryHighlight:   "#89B4FA",
	SecondaryHighlight: "#CBA6F7",
	InactiveBorder:     "#45475A",
	Success:            "#A6E3A1",
	Error:              "#F38BA8",
	Warning:            "#F9E2AF",
	Info:               "#89DCEB",
	Muted:              "#585B70",
	VisualizerPlayed:   []RGB{{R: 137, G: 180, B: 250}, {R: 203, G: 166, B: 247}, {R: 245, G: 194, B: 231}},
	VisualizerUnplayed: []RGB{{R: 49, G: 50, B: 68}, {R: 69, G: 71, B: 90}, {R: 88, G: 91, B: 112}},
	EqualizerChar:      '●',
	EqualizerBg:        "#313244",
}

var nord = Theme{
	Name:               "nord",
	HeaderTitle:        "▶ GoYT",
	Background:         "#2E3440",
	Surface:            "#3B4252",
	TextPrimary:        "#ECEFF4",
	TextSecondary:      "#D8DEE9",
	TextOnAccent:       "#2E3440",
	PrimaryHighlight:   "#88C0D0",
	SecondaryHighlight: "#81A1C1",
	InactiveBorder:     "#4C566A",
	Success:            "#A3BE8C",
	Error:              "#BF616A",
	Warning:            "#EBCB8B",
	Info:               "#5E81AC",
	Muted:              "#4C566A",
	VisualizerPlayed:   []RGB{{R: 136, G: 192, B: 208}, {R: 129, G: 161, B: 193}, {R: 94, G: 129, B: 172}},
	VisualizerUnplayed: []RGB{{R: 59, G: 66, B: 82}, {R: 67, G: 76, B: 94}, {R: 76, G: 86, B: 106}},
	EqualizerChar:      '●',
	EqualizerBg:        "#3B4252",
}

var dracula = Theme{
	Name:               "dracula",
	HeaderTitle:        "▶ GoYT",
	Background:         "#282A36",
	Surface:            "#44475A",
	TextPrimary:        "#F8F8F2",
	TextSecondary:      "#BFBFBF",
	TextOnAccent:       "#282A36",
	PrimaryHighlight:   "#BD93F9",
	SecondaryHighlight: "#FF79C6",
	InactiveBorder:     "#6272A4",
	Success:            "#50FA7B",
	Error:              "#FF5555",
	Warning:            "#F1FA8C",
	Info:               "#8BE9FD",
	Muted:              "#6272A4",
	VisualizerPlayed:   []RGB{{R: 189, G: 147, B: 249}, {R: 255, G: 121, B: 198}, {R: 255, G: 85, B: 85}},
	VisualizerUnplayed: []RGB{{R: 68, G: 71, B: 90}, {R: 98, G: 114, B: 164}, {R: 68, G: 71, B: 90}},
	EqualizerChar:      '●',
	EqualizerBg:        "#44475A",
}
