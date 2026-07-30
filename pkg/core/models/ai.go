package models

// AIConfig stores AI commit message provider settings.
// APIKey is always masked or empty when sent to the frontend.
type AIConfig struct {
	Provider  string           `json:"provider"`
	APIKey    string           `json:"api_key"` // masked or empty
	Model     string           `json:"model"`
	Endpoint  string           `json:"endpoint,omitempty"`
	Providers []ProviderStatus `json:"providers,omitempty"` // per-provider key status
}

// ProviderStatus describes a single provider's key state for the frontend.
// Only masked key info is exposed — never the plaintext key.
type ProviderStatus struct {
	Provider  string `json:"provider"`
	HasKey    bool   `json:"has_key"`
	KeyMasked string `json:"key_masked"` // e.g. "sk-...f3a2" or ""
	Model     string `json:"model,omitempty"`
	Endpoint  string `json:"endpoint,omitempty"`
}
