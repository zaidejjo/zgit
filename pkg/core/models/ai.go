package models

// AIConfig stores AI commit message provider settings.
type AIConfig struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
	Model    string `json:"model"`
	Endpoint string `json:"endpoint,omitempty"`
}
