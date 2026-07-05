package json

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"goyt/internal/domain/model"
)

// ConfigJSON matches the format of config.json
type ConfigJSON struct {
	Cookie              string `json:"cookie"`
	ThemeName           string `json:"theme_name,omitempty"`
	EnableNotifications *bool  `json:"enable_notifications,omitempty"`
	Theme               *struct {
		PrimaryHighlight   string   `json:"primary_highlight"`
		SecondaryHighlight string   `json:"secondary_highlight"`
		InactiveBorder     string   `json:"inactive_border"`
		Background         string   `json:"background,omitempty"`
		Surface            string   `json:"surface,omitempty"`
		TextPrimary        string   `json:"text_primary,omitempty"`
		TextSecondary      string   `json:"text_secondary,omitempty"`
		TextOnAccent       string   `json:"text_on_accent,omitempty"`
		Success            string   `json:"success,omitempty"`
		Error              string   `json:"error,omitempty"`
		Warning            string   `json:"warning,omitempty"`
		Info               string   `json:"info,omitempty"`
		Muted              string   `json:"muted,omitempty"`
		EqualizerBg        string   `json:"equalizer_bg,omitempty"`
		HeaderTitle        string   `json:"header_title,omitempty"`
		VisualizerPlayed   []string `json:"visualizer_played"`
		VisualizerUnplayed []string `json:"visualizer_unplayed"`
		EqualizerChar      string   `json:"equalizer_char"`
	} `json:"theme,omitempty"`
}

type JsonConfigAdapter struct {
	filePath string
}

func NewJsonConfigAdapter() (*JsonConfigAdapter, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".config", "goyt", "config.json")
	return &JsonConfigAdapter{filePath: path}, nil
}

func (a *JsonConfigAdapter) LoadCookie() (string, error) {
	cfg, err := a.readConfig()
	if err != nil {
		return "", err
	}
	return cfg.Cookie, nil
}

func (a *JsonConfigAdapter) LoadTheme() (*model.Theme, error) {
	cfg, err := a.readConfig()
	if err != nil {
		return nil, err
	}

	// Start from a named preset (default: ios-dark)
	presetName := cfg.ThemeName
	if presetName == "" {
		presetName = model.DefaultThemeName
	}
	base, ok := model.PresetThemes[presetName]
	if !ok {
		base = model.PresetThemes[model.DefaultThemeName]
	}
	theme := base // copy

	if cfg.Theme == nil {
		return &theme, nil
	}

	t := cfg.Theme

	// Overlay string fields
	if t.PrimaryHighlight != "" {
		theme.PrimaryHighlight = t.PrimaryHighlight
	}
	if t.SecondaryHighlight != "" {
		theme.SecondaryHighlight = t.SecondaryHighlight
	}
	if t.InactiveBorder != "" {
		theme.InactiveBorder = t.InactiveBorder
	}
	if t.Background != "" {
		theme.Background = t.Background
	}
	if t.Surface != "" {
		theme.Surface = t.Surface
	}
	if t.TextPrimary != "" {
		theme.TextPrimary = t.TextPrimary
	}
	if t.TextSecondary != "" {
		theme.TextSecondary = t.TextSecondary
	}
	if t.TextOnAccent != "" {
		theme.TextOnAccent = t.TextOnAccent
	}
	if t.Success != "" {
		theme.Success = t.Success
	}
	if t.Error != "" {
		theme.Error = t.Error
	}
	if t.Warning != "" {
		theme.Warning = t.Warning
	}
	if t.Info != "" {
		theme.Info = t.Info
	}
	if t.Muted != "" {
		theme.Muted = t.Muted
	}
	if t.EqualizerBg != "" {
		theme.EqualizerBg = t.EqualizerBg
	}
	if t.HeaderTitle != "" {
		theme.HeaderTitle = t.HeaderTitle
	}

	parseHexColor := func(s string) (model.RGB, error) {
		var r, g, b int
		s = strings.TrimPrefix(s, "#")
		if len(s) == 3 {
			_, err := fmt.Sscanf(s, "%1x%1x%1x", &r, &g, &b)
			if err != nil {
				return model.RGB{}, err
			}
			return model.RGB{R: r * 17, G: g * 17, B: b * 17}, nil
		} else if len(s) == 6 {
			_, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b)
			if err != nil {
				return model.RGB{}, err
			}
			return model.RGB{R: r, G: g, B: b}, nil
		}
		return model.RGB{}, fmt.Errorf("invalid hex color: %s", s)
	}

	if len(t.VisualizerPlayed) > 0 {
		var played []model.RGB
		for _, hexStr := range t.VisualizerPlayed {
			if rgb, err := parseHexColor(hexStr); err == nil {
				played = append(played, rgb)
			}
		}
		if len(played) > 0 {
			theme.VisualizerPlayed = played
		}
	}

	if len(t.VisualizerUnplayed) > 0 {
		var unplayed []model.RGB
		for _, hexStr := range t.VisualizerUnplayed {
			if rgb, err := parseHexColor(hexStr); err == nil {
				unplayed = append(unplayed, rgb)
			}
		}
		if len(unplayed) > 0 {
			theme.VisualizerUnplayed = unplayed
		}
	}

	if t.EqualizerChar != "" {
		runes := []rune(t.EqualizerChar)
		if len(runes) > 0 {
			theme.EqualizerChar = runes[0]
		}
	}

	return &theme, nil
}

func (a *JsonConfigAdapter) readConfig() (*ConfigJSON, error) {
	dir := filepath.Dir(a.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	if _, err := os.Stat(a.filePath); os.IsNotExist(err) {
		cfg := &ConfigJSON{}
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(a.filePath, data, 0644); err != nil {
			return nil, err
		}
		return cfg, nil
	}

	data, err := os.ReadFile(a.filePath)
	if err != nil {
		return nil, err
	}

	var cfg ConfigJSON
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (a *JsonConfigAdapter) LoadNotificationsEnabled() (bool, error) {
	cfg, err := a.readConfig()
	if err != nil {
		return true, nil // default to true on error
	}
	if cfg.EnableNotifications != nil {
		return *cfg.EnableNotifications, nil
	}
	return true, nil // default to true if unspecified
}
