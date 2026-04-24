package vscode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Server struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Type         string   `json:"type"`
	Address      string   `json:"address"`
	Port         int      `json:"port"`
	BuildVersion string   `json:"buildVersion"`
	Environments []string `json:"environments"`
	// v2.x usa "environment", v1.x usa "currentEnvironment"
	Environment        string `json:"environment"`
	CurrentEnvironment string `json:"currentEnvironment"`
}

type ServersConfig struct {
	Includes        []string `json:"includes"`
	ConnectedServer *Server  `json:"connectedServer"`
	// v2.x usa "configurations", v1.x usa "servers"
	Configurations      []Server `json:"configurations"`
	Servers             []Server `json:"servers"`
	LastConnectedServer string   `json:"lastConnectedServer"`
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
		normalizeServer(config.ConnectedServer)
		return config.ConnectedServer, nil
	}

	// Unifica lista de servidores (v2.x = configurations, v1.x = servers)
	allServers := append(config.Configurations, config.Servers...)

	// Prioridade 2: servidor com ambiente selecionado
	for i := range allServers {
		normalizeServer(&allServers[i])
		if allServers[i].CurrentEnvironment != "" {
			return &allServers[i], nil
		}
	}

	// Prioridade 3: primeiro servidor da lista
	if len(allServers) > 0 && allServers[0].Address != "" {
		return &allServers[0], nil
	}

	return nil, fmt.Errorf(
		"nenhum servidor ativo encontrado.\n" +
			"Abra o VS Code, conecte-se a um servidor na extensão TOTVS e tente novamente.",
	)
}

// normalizeServer garante que CurrentEnvironment esteja preenchido (compatibilidade v1/v2)
func normalizeServer(s *Server) {
	if s.CurrentEnvironment == "" && s.Environment != "" {
		s.CurrentEnvironment = s.Environment
	}
}

func serversJSONPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("não foi possível localizar o diretório do usuário: %w", err)
	}

	appData := os.Getenv("APPDATA")
	if appData == "" {
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	// Caminhos por versão da extensão TOTVS (do mais recente ao mais antigo)
	candidates := []string{
		filepath.Join(home, ".totvsls", "servers.json"),                                                          // v2.x
		filepath.Join(appData, "Code", "User", "globalStorage", "totvs.tds-vscode", "servers.json"),             // v1.x
		filepath.Join(appData, "Code - Insiders", "User", "globalStorage", "totvs.tds-vscode", "servers.json"),  // VS Code Insiders
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf(
		"arquivo servers.json não encontrado.\n\n"+
			"Isso indica que a extensão 'TOTVS Developer Studio' não está configurada.\n"+
			"Abra o VS Code, instale a extensão TOTVS e configure ao menos um servidor.\n\n"+
			"Caminhos verificados:\n  - %s",
		strings.Join(candidates, "\n  - "),
	)
}
