package configs

import (
	"fmt"
	"os"
	"path/filepath"
)

// DefaultDataDir returns the default data dir "~/.blindbit-desktop/"
// if homedir is not found falls back to current directory "."
func DefaultDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".blindbit-desktop")
	fmt.Printf("Data directory: %s\n", dataDir)
	return dataDir
}

const (
	DefaultOracleAddress = "127.0.0.1:7001"
	DefaultMinimumAmount = 546
	DefaultLabelCount    = 0
)
