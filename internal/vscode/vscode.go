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
	// Campos alternativos em versões mais antigas da extensão
	SmartClientPath string `json:"smartClientPath"`
}

type ServersConfig struct {
	Includes        []string `json:"includes"`
	ConnectedServer *Server  `json:"connectedServer"`
	Servers         []Server `json:"servers"`
	// Campo presente em algumas versões
	LastConnectedServer string `json:"lastConnectedServer"`
}

func LoadServersConfig() (*ServersConfig, error) {
	path, err := serversJSONPath()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf(
			"arquivo servers.json não encontrado.\n\n"+
				"Isso indica que a extensão 'TOTVS Developer Studio' não está configurada.\n"+
				"Abra o VS Code, instale a extensão TOTVS e configure ao menos um servidor.\n\n"+
				"Caminho esperado: %s", path,
		)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("não foi possível ler servers.json: %w", err)
	}

	var config ServersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf(
			"erro ao interpretar servers.json — o formato pode ser incompatível com esta versão do tlpkg: %w", err,
		)
	}

	if len(config.Servers) == 0 && config.ConnectedServer == nil {
		return nil, fmt.Errorf(
			"nenhum servidor encontrado no servers.json.\n" +
				"Configure ao menos um servidor na extensão TOTVS do VS Code e tente novamente.",
		)
	}

	return &config, nil
}

func ActiveServer(config *ServersConfig) (*Server, error) {
	// Prioridade 1: servidor conectado no momento
	if config.ConnectedServer != nil && config.ConnectedServer.Address != "" {
		return config.ConnectedServer, nil
	}

	// Prioridade 2: servidor com ambiente selecionado
	for i, s := range config.Servers {
		if s.CurrentEnvironment != "" {
			return &config.Servers[i], nil
		}
	}

	// Prioridade 3: primeiro servidor da lista (fallback)
	if len(config.Servers) > 0 {
		s := &config.Servers[0]
		if s.Address != "" {
			return s, nil
		}
	}

	return nil, fmt.Errorf(
		"nenhum servidor ativo encontrado.\n" +
			"Abra o VS Code, conecte-se a um servidor na extensão TOTVS e tente novamente.",
	)
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
