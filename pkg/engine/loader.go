package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadPolicyConfig reads and parses a policy file (JSON or YAML)
func LoadPolicyConfig(path string) (*PolicyConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config PolicyConfig
	ext := strings.ToLower(filepath.Ext(path))

	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, err
		}
	} else {
		// Default to JSON
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, err
		}
	}

	return &config, nil
}
