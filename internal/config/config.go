package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type TlpkgConfig struct {
	Server      string   `json:"server"`
	Port        int      `json:"port"`
	Environment string   `json:"environment"`
	Build       string   `json:"build"`
	Includes    []string `json:"includes"`
}

func ConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar o diretório do usuário: %w", err)
	}
	return filepath.Join(home, ".tlpkg", "config.json"), nil
}

func Load() (*TlpkgConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("erro ao ler configuração do tlpkg: %w", err)
	}

	var cfg TlpkgConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("configuração inválida em %s: %w", path, err)
	}

	return &cfg, nil
}

func Save(cfg *TlpkgConfig) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("erro ao criar diretório de configuração: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("erro ao salvar configuração: %w", err)
	}

	return nil
}

func Exists() bool {
	path, err := ConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
