package cmd

import (
	"os"
	"path/filepath"
)

// GetDefaultConfigPath returns the default configuration file path.
func GetDefaultConfigPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, defaultConfigName)
}