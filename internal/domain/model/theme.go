package model

type RGB struct {
	R, G, B int
}

type Theme struct {
	PrimaryHighlight   string
	SecondaryHighlight string
	InactiveBorder     string
	VisualizerPlayed   []RGB
	VisualizerUnplayed []RGB
	EqualizerChar      rune
}
