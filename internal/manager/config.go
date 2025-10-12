package manager

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// setDefaultConfig sets default configuration values
func setDefaultConfig(config *viper.Viper) {
	config.SetDefault("network", DefaultNetwork)
	config.SetDefault("oracle_url", "https://silentpayments.dev/blindbit/mainnet") // Default for mainnet
	// config.SetDefault("http_port", 8080)
	// todo: this is probably not relevant anymore
	// config.SetDefault("private_mode", false)
	config.SetDefault("dust_limit", 546)
	config.SetDefault("label_count", 0)
	// Default birth height for mainnet
	config.SetDefault("birth_height", 900000)
	config.SetDefault("min_change_amount", 546)
}

// getDataDir returns the appropriate data directory for the application
func getDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".blindbit-desktop")
	fmt.Printf("Data directory: %s\n", dataDir)
	return dataDir
}

// initializeConfig initializes the configuration with defaults and loads existing config
func initializeConfig(dataDir string) (*viper.Viper, error) {
	// Initialize configuration
	config := viper.New()
	config.SetConfigName("blindbit")
	config.SetConfigType("toml")
	config.AddConfigPath(dataDir)

	// Set default values
	setDefaultConfig(config)

	// Load existing config if it exists
	if err := config.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file doesn't exist, create default one
		if err := config.WriteConfigAs(filepath.Join(dataDir, "blindbit.toml")); err != nil {
			fmt.Printf("Error writing default config: %v\n", err)
			return nil, fmt.Errorf("failed to write default config: %w", err)
		}
		fmt.Println("Default config file created")
	} else {
		fmt.Println("Existing config loaded")
	}

	return config, nil
}
