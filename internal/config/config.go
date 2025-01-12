package config

import (
	"encoding/json"
	"fmt"
	"os"
)

const configFileName = "/.config/gatorconfig.json"

type Config struct {
	DbURL string `json:"db_url"`
	User  string `json:"user"`
}

func (cfg *Config) SetUser(username string) bool {
	cfg.User = username

	err := cfg.Write()

	if err != nil {
		fmt.Println(err)

		return false
	}

	return true
}

func (cfg *Config) Write() error {
	configFilePath, err := getConfigFilePath()

	if err != nil {
		return err
	}

	file, err := os.Create(configFilePath)

	if err != nil {
		return err
	}

	defer file.Close()

	err = json.NewEncoder(file).Encode(cfg)

	if err != nil {
		return err
	}

	return nil
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}

	return homeDir + configFileName, nil
}

func Read() (Config, error) {
	var cfg Config

	configFilePath, err := getConfigFilePath()

	if err != nil {
		return cfg, err
	}

	file, err := os.Open(configFilePath)

	if err != nil {
		return cfg, err
	}

	defer file.Close()

	err = json.NewDecoder(file).Decode(&cfg)

	if err != nil {
		return cfg, err
	}

	return cfg, nil
}
