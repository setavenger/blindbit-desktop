package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const settingsFilename = "settings.json"

// Settings holds wallet connection preferences that are stored separately
// from the encrypted wallet blob so they can be read without decryption.
type Settings struct {
	OracleAddress string `json:"oracle_address"`
	OracleUseTLS  bool   `json:"oracle_use_tls"`
}

// SaveSettings writes the settings to <datadir>/settings.json.
func SaveSettings(datadir string, s *Settings) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}
	path := filepath.Join(datadir, settingsFilename)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}
	return nil
}

// LoadSettings reads <datadir>/settings.json.
// Returns nil, nil when the file does not exist (first run / migration).
func LoadSettings(datadir string) (*Settings, error) {
	path := filepath.Join(datadir, settingsFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read settings file: %w", err)
	}
	var s Settings
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("failed to parse settings file: %w", err)
	}
	return &s, nil
}
