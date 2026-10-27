package app

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// UserConfig represents the minimal user configuration file.
type UserConfig struct {
	PreferredModel string              `yaml:"preferred_model,omitempty"`
	Fallbacks      map[string][]string `yaml:"fallbacks,omitempty"` // model -> []providerID
}

// SaveFallbackToConfig appends or saves a discovered working fallback provider to soundcheck.yaml.
func SaveFallbackToConfig(configPath string, modelName string, providerID string) error {
	cleanPath := filepath.Clean(configPath)
	if cleanPath == "" || cleanPath == "." {
		cleanPath = "soundcheck.yaml"
	}

	cfg := &UserConfig{
		Fallbacks: make(map[string][]string),
	}

	// Read existing if present
	if data, err := os.ReadFile(cleanPath); err == nil { //nolint:gosec
		_ = yaml.Unmarshal(data, cfg)
	}

	if cfg.Fallbacks == nil {
		cfg.Fallbacks = make(map[string][]string)
	}

	existingList := cfg.Fallbacks[modelName]
	alreadyExists := false
	for _, p := range existingList {
		if p == providerID {
			alreadyExists = true
			break
		}
	}

	if !alreadyExists {
		cfg.Fallbacks[modelName] = append(existingList, providerID)
	}

	if cfg.PreferredModel == "" {
		cfg.PreferredModel = modelName
	}

	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return os.WriteFile(cleanPath, out, 0600) //nolint:gosec
}
