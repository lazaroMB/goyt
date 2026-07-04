package model

type RGB struct {
	R, G, B int
}

type Theme struct {
	// Identity
	Name        string
	HeaderTitle string

	// Core Palette
	Background    string
	Surface       string
	TextPrimary   string
	TextSecondary string
	TextOnAccent  string

	// Accent Colors
	PrimaryHighlight   string
	SecondaryHighlight string
	InactiveBorder     string

	// Semantic Colors
	Success string
	Error   string
	Warning string
	Info    string
	Muted   string

	// Visualizer
	VisualizerPlayed   []RGB
	VisualizerUnplayed []RGB
	EqualizerChar      rune
	EqualizerBg        string
}
