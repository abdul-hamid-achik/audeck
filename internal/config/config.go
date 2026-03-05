package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Profile represents a saved audio device configuration.
type Profile struct {
	Name              string  `json:"name"`
	DefaultOutput     string  `json:"default_output,omitempty"`
	DefaultInput      string  `json:"default_input,omitempty"`
	OutputVolume      float32 `json:"output_volume,omitempty"`
	InputVolume       float32 `json:"input_volume,omitempty"`
	Muted             bool    `json:"muted,omitempty"`
}

// Keybindings holds custom keyboard shortcuts.
type Keybindings struct {
	Quit           string `json:"quit,omitempty"`
	NavUp          string `json:"nav_up,omitempty"`
	NavDown        string `json:"nav_down,omitempty"`
	NavTop         string `json:"nav_top,omitempty"`
	NavBottom      string `json:"nav_bottom,omitempty"`
	SetDefault     string `json:"set_default,omitempty"`
	VolumeUp       string `json:"volume_up,omitempty"`
	VolumeDown     string `json:"volume_down,omitempty"`
	VolumeFineUp   string `json:"volume_fine_up,omitempty"`
	VolumeFineDown string `json:"volume_fine_down,omitempty"`
	ToggleMute     string `json:"toggle_mute,omitempty"`
	TabOutput      string `json:"tab_output,omitempty"`
	TabInput       string `json:"tab_input,omitempty"`
}

// DefaultKeybindings returns the default keybindings.
func DefaultKeybindings() Keybindings {
	return Keybindings{
		Quit:           "q",
		NavUp:          "k",
		NavDown:        "j",
		NavTop:         "g",
		NavBottom:      "G",
		SetDefault:     "enter",
		VolumeUp:       "l",
		VolumeDown:     "h",
		VolumeFineUp:   "+",
		VolumeFineDown: "-",
		ToggleMute:     "m",
		TabOutput:      "1",
		TabInput:       "2",
	}
}

// Merge merges custom keybindings with defaults.
func (k Keybindings) Merge(defaults Keybindings) Keybindings {
	if k.Quit != "" {
		defaults.Quit = k.Quit
	}
	if k.NavUp != "" {
		defaults.NavUp = k.NavUp
	}
	if k.NavDown != "" {
		defaults.NavDown = k.NavDown
	}
	if k.NavTop != "" {
		defaults.NavTop = k.NavTop
	}
	if k.NavBottom != "" {
		defaults.NavBottom = k.NavBottom
	}
	if k.SetDefault != "" {
		defaults.SetDefault = k.SetDefault
	}
	if k.VolumeUp != "" {
		defaults.VolumeUp = k.VolumeUp
	}
	if k.VolumeDown != "" {
		defaults.VolumeDown = k.VolumeDown
	}
	if k.VolumeFineUp != "" {
		defaults.VolumeFineUp = k.VolumeFineUp
	}
	if k.VolumeFineDown != "" {
		defaults.VolumeFineDown = k.VolumeFineDown
	}
	if k.ToggleMute != "" {
		defaults.ToggleMute = k.ToggleMute
	}
	if k.TabOutput != "" {
		defaults.TabOutput = k.TabOutput
	}
	if k.TabInput != "" {
		defaults.TabInput = k.TabInput
	}
	return defaults
}

// Theme represents a color theme configuration.
type Theme string

const (
	ThemeCatppuccinMocha Theme = "catppuccin-mocha"
	ThemeCatppuccinMacchiato Theme = "catppuccin-macchiato"
	ThemeCatppuccinFrappe Theme = "catppuccin-frappe"
	ThemeCatppuccinLatte Theme = "catppuccin-latte"
	ThemeDracula Theme = "dracula"
	ThemeNord Theme = "nord"
	ThemeGruvbox Theme = "gruvbox"
)

// ThemeColors represents a color palette for a theme.
type ThemeColors struct {
	Base       string
	Mantle     string
	Crust      string
	Surface0   string
	Surface1   string
	Surface2   string
	Overlay0   string
	Overlay1   string
	Text       string
	Subtext0   string
	Subtext1   string
	Lavender   string
	Blue       string
	Sapphire   string
	Teal       string
	Green      string
	Yellow     string
	Peach      string
	Maroon     string
	Red        string
	Mauve      string
	Pink       string
}

// GetThemeColors returns the color palette for a theme.
func (t Theme) GetColors() ThemeColors {
	switch t {
	case ThemeCatppuccinMacchiato:
		return ThemeColors{
			Base: "#24273a", Mantle: "#1e2030", Crust: "#181926",
			Surface0: "#363a4f", Surface1: "#494d64", Surface2: "#5b6078",
			Overlay0: "#6e738d", Overlay1: "#8087a2",
			Text: "#cad3f5", Subtext0: "#a5adcb", Subtext1: "#b8c0e0",
			Lavender: "#b7bdf8", Blue: "#8aadf4", Sapphire: "#7dc4e4",
			Teal: "#8bd5ca", Green: "#a6da95", Yellow: "#eed49f",
			Peach: "#f5a97f", Maroon: "#ee99a0", Red: "#ed8796",
			Mauve: "#c6a0f6", Pink: "#f5bde6",
		}
	case ThemeCatppuccinFrappe:
		return ThemeColors{
			Base: "#303446", Mantle: "#292c3c", Crust: "#232634",
			Surface0: "#414559", Surface1: "#51576d", Surface2: "#626880",
			Overlay0: "#737994", Overlay1: "#838ba7",
			Text: "#c6d0f5", Subtext0: "#a5adce", Subtext1: "#b5bfe2",
			Lavender: "#babbf1", Blue: "#8caaee", Sapphire: "#75c4e4",
			Teal: "#81c8be", Green: "#a6d189", Yellow: "#e5c890",
			Peach: "#ef9f76", Maroon: "#ea999c", Red: "#e78284",
			Mauve: "#ca9ee6", Pink: "#f4b8e4",
		}
	case ThemeCatppuccinLatte:
		return ThemeColors{
			Base: "#eff1f5", Mantle: "#e6e9ef", Crust: "#dce0e8",
			Surface0: "#ccd0da", Surface1: "#bcc0cc", Surface2: "#acb0be",
			Overlay0: "#9ca0b0", Overlay1: "#8c8fa1",
			Text: "#4c4f69", Subtext0: "#6c6f85", Subtext1: "#5c5f77",
			Lavender: "#7287fd", Blue: "#1e66f5", Sapphire: "#209fb5",
			Teal: "#179299", Green: "#40a02b", Yellow: "#df8e1d",
			Peach: "#fe640b", Maroon: "#e64553", Red: "#d20f39",
			Mauve: "#8839ef", Pink: "#ea76cb",
		}
	case ThemeDracula:
		return ThemeColors{
			Base: "#282a36", Mantle: "#21222c", Crust: "#191a21",
			Surface0: "#44475a", Surface1: "#6272a4", Surface2: "#7970a9",
			Overlay0: "#8be9fd", Overlay1: "#bd93f9",
			Text: "#f8f8f2", Subtext0: "#bfbfbf", Subtext1: "#e0e0e0",
			Lavender: "#bd93f9", Blue: "#8be9fd", Sapphire: "#8be9fd",
			Teal: "#8be9fd", Green: "#50fa7b", Yellow: "#f1fa8c",
			Peach: "#ffb86c", Maroon: "#ff79c6", Red: "#ff5555",
			Mauve: "#bd93f9", Pink: "#ff79c6",
		}
	case ThemeNord:
		return ThemeColors{
			Base: "#2e3440", Mantle: "#272c36", Crust: "#1f242b",
			Surface0: "#3b4252", Surface1: "#434c5e", Surface2: "#4c566a",
			Overlay0: "#d8dee9", Overlay1: "#e5e9f0",
			Text: "#eceff4", Subtext0: "#e5e9f0", Subtext1: "#eceff4",
			Lavender: "#b48ead", Blue: "#88c0d0", Sapphire: "#81a1c1",
			Teal: "#8fbcbb", Green: "#a3be8c", Yellow: "#ebcb8b",
			Peach: "#d08770", Maroon: "#bf616a", Red: "#bf616a",
			Mauve: "#b48ead", Pink: "#b48ead",
		}
	case ThemeGruvbox:
		return ThemeColors{
			Base: "#282828", Mantle: "#1d2021", Crust: "#16191a",
			Surface0: "#3c3836", Surface1: "#504945", Surface2: "#665c54",
			Overlay0: "#bdae93", Overlay1: "#d5c4a1",
			Text: "#ebdbb2", Subtext0: "#d5c4a1", Subtext1: "#ebdbb2",
			Lavender: "#d3869b", Blue: "#83a598", Sapphire: "#83a598",
			Teal: "#8ec07c", Green: "#98971a", Yellow: "#d79921",
			Peach: "#d65d0e", Maroon: "#cc241d", Red: "#cc241d",
			Mauve: "#d3869b", Pink: "#d3869b",
		}
	default: // Catppuccin Mocha (default)
		return ThemeColors{
			Base: "#1e1e2e", Mantle: "#181825", Crust: "#11111b",
			Surface0: "#313244", Surface1: "#45475a", Surface2: "#585b70",
			Overlay0: "#6c7086", Overlay1: "#7f849c",
			Text: "#cdd6f4", Subtext0: "#a6adc8", Subtext1: "#bac2de",
			Lavender: "#b4befe", Blue: "#89b4fa", Sapphire: "#74c7ec",
			Teal: "#94e2d5", Green: "#a6e3a1", Yellow: "#f9e2af",
			Peach: "#fab387", Maroon: "#eba0ac", Red: "#f38ba8",
			Mauve: "#cba6f7", Pink: "#f5c2e7",
		}
	}
}

// Config holds user-persisted preferences.
type Config struct {
	// DefaultOutputDevice is the preferred default output device UID.
	DefaultOutputDevice string `json:"default_output_device,omitempty"`
	// DefaultInputDevice is the preferred default input device UID.
	DefaultInputDevice string `json:"default_input_device,omitempty"`
	// Profiles is a list of saved audio profiles.
	Profiles []Profile `json:"profiles,omitempty"`
	// ActiveProfile is the name of the currently active profile.
	ActiveProfile string `json:"active_profile,omitempty"`
	// Keybindings holds custom keyboard shortcuts.
	Keybindings Keybindings `json:"keybindings,omitempty"`
	// Theme is the selected color theme.
	Theme Theme `json:"theme,omitempty"`
}

// configPath allows overriding the config file path for testing.
var configPath string

// Path returns the default config file path (~/.config/audeck/config.json).
func Path() (string, error) {
	if configPath != "" {
		return configPath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "audeck", "config.json"), nil
}

// Load reads the config from disk. Returns a zero Config if the file
// does not exist.
func Load() (Config, error) {
	p, err := Path()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return Config{}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var c Config
	return c, json.Unmarshal(data, &c)
}

// Save writes the config to disk, creating directories as needed.
func (c Config) Save() error {
	p, err := Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// AddProfile adds or updates a profile.
func (c *Config) AddProfile(profile Profile) {
	for i, p := range c.Profiles {
		if p.Name == profile.Name {
			c.Profiles[i] = profile
			return
		}
	}
	c.Profiles = append(c.Profiles, profile)
}

// GetProfile retrieves a profile by name.
func (c *Config) GetProfile(name string) (Profile, bool) {
	for _, p := range c.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

// DeleteProfile removes a profile by name.
func (c *Config) DeleteProfile(name string) bool {
	for i, p := range c.Profiles {
		if p.Name == name {
			c.Profiles = append(c.Profiles[:i], c.Profiles[i+1:]...)
			return true
		}
	}
	return false
}

// ListProfiles returns all profile names.
func (c *Config) ListProfiles() []string {
	names := make([]string, len(c.Profiles))
	for i, p := range c.Profiles {
		names[i] = p.Name
	}
	return names
}
