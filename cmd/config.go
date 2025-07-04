package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type User struct {
	Username string `mapstructure:"username"`
	Token    string `mapstructure:"token"`
}

type DefaultSettings struct {
	Count  int      `mapstructure:"count"`
	Since  string   `mapstructure:"since"`
	Filter []string `mapstructure:"filter"`
}

type Config struct {
	Users           []User          `mapstructure:"users"`
	DefaultSettings DefaultSettings `mapstructure:"default_settings"`
}

func loadConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "gitfamous")
	configFile := filepath.Join(configDir, "config.yml")

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s", configFile)
	}

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(configDir)

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate config
	if len(config.Users) == 0 {
		return nil, fmt.Errorf("no users defined in config file")
	}

	for i, user := range config.Users {
		if user.Username == "" {
			return nil, fmt.Errorf("user %d has empty username", i+1)
		}
	}

	return &config, nil
}

func configExists() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configFile := filepath.Join(homeDir, ".config", "gitfamous", "config.yml")
	_, err = os.Stat(configFile)
	return err == nil
}
