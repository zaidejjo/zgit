package models

// UserPreferences holds all user-customizable settings persisted to disk.
type UserPreferences struct {
	Appearance  AppearanceConfig  `json:"appearance"`
	Keybindings map[string]string `json:"keybindings"`
}

// AppearanceConfig controls theme-level visual customization.
type AppearanceConfig struct {
	Theme       string `json:"theme"`
	AccentColor string `json:"accent_color"`
	Brightness  int    `json:"brightness"`
}
