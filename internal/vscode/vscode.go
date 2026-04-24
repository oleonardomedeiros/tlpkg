package vscode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Server struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Type               string   `json:"type"`
	Address            string   `json:"address"`
	Port               int      `json:"port"`
	BuildVersion       string   `json:"buildVersion"`
	Environments       []string `json:"environments"`
	CurrentEnvironment string   `json:"currentEnvironment"`
}

type ServersConfig struct {
	Includes        []string `json:"includes"`
	ConnectedServer *Server  `json:"connectedServer"`
	Servers         []Server `json:"servers"`
}

func LoadServersConfig() (*ServersConfig, error) {
	path, err := serversJSONPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler servers.json: %w\ncaminho: %s", err, path)
	}

	var config ServersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("erro ao interpretar servers.json: %w", err)
	}

	return &config, nil
}

func ActiveServer(config *ServersConfig) (*Server, error) {
	if config.ConnectedServer != nil && config.ConnectedServer.Address != "" {
		return config.ConnectedServer, nil
	}

	for i, s := range config.Servers {
		if s.CurrentEnvironment != "" {
			return &config.Servers[i], nil
		}
	}

	return nil, fmt.Errorf("nenhum servidor ativo encontrado no VS Code\nconfigure a conexão na extensão TOTVS e tente novamente")
}

func serversJSONPath() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("não foi possível localizar o diretório do usuário: %w", err)
		}
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	return filepath.Join(appData, "Code", "User", "globalStorage", "totvs.tds-vscode", "servers.json"), nil
}
