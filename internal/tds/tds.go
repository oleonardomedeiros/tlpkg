package tds

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/oleonardomedeiros/tlpkg/internal/config"
	"github.com/oleonardomedeiros/tlpkg/internal/vscode"
)

type Client struct {
	name        string
	address     string
	port        int
	environment string
	build       string
	includes    []string
	advplsPath  string
}

// NewClient tenta VS Code primeiro, cai para config própria se não encontrar
func NewClient() (*Client, error) {
	advplsPath, err := findAdvpls()
	if err != nil {
		return nil, err
	}

	// Tentativa 1: lê do VS Code
	vsConfig, vsErr := vscode.LoadServersConfig()
	if vsErr == nil {
		server, sErr := vscode.ActiveServer(vsConfig)
		if sErr == nil {
			return &Client{
				name:        server.Name,
				address:     server.Address,
				port:        server.Port,
				environment: server.CurrentEnvironment,
				build:       server.BuildVersion,
				includes:    vsConfig.Includes,
				advplsPath:  advplsPath,
			}, nil
		}
	}

	// Tentativa 2: config própria do tlpkg
	cfg, cfgErr := config.Load()
	if cfgErr == nil && cfg != nil && cfg.Server != "" {
		return &Client{
			name:        cfg.Server,
			address:     cfg.Server,
			port:        cfg.Port,
			environment: cfg.Environment,
			build:       cfg.Build,
			includes:    cfg.Includes,
			advplsPath:  advplsPath,
		}, nil
	}

	// Nenhuma fonte funcionou — orienta o usuário
	return nil, fmt.Errorf(
		"não foi possível detectar a configuração do AppServer.\n\n" +
			"Opção 1 — Configure na extensão TOTVS do VS Code e certifique-se de estar conectado.\n" +
			"Opção 2 — Configure manualmente:\n\n" +
			"  tlpkg config\n",
	)
}

func (c *Client) ServerInfo() string {
	return fmt.Sprintf("servidor: %s (%s:%d) | ambiente: %s",
		c.name, c.address, c.port, c.environment,
	)
}

func (c *Client) Compile(filePath string, recompile bool) error {
	return c.runTDSCli("compile", filePath, recompile)
}

func (c *Client) Delete(filePath string) error {
	return c.runTDSCli("delete", filePath, false)
}

func (c *Client) runTDSCli(action, filePath string, recompile bool) error {
	args := []string{
		"tds-cli",
		action,
		"serverType=AdvPL",
		fmt.Sprintf("server=%s", c.address),
		fmt.Sprintf("port=%d", c.port),
		fmt.Sprintf("build=%s", c.build),
		fmt.Sprintf("environment=%s", c.environment),
		fmt.Sprintf("program=%s", filePath),
	}

	if len(c.includes) > 0 {
		args = append(args, fmt.Sprintf("includes=%s", strings.Join(c.includes, ";")))
	}

	if action == "compile" {
		recompileVal := "f"
		if recompile {
			recompileVal = "t"
		}
		args = append(args, fmt.Sprintf("recompile=%s", recompileVal))
	}

	cmd := exec.Command(c.advplsPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("TDS-CLI retornou erro: %w", err)
	}

	return nil
}

func findAdvpls() (string, error) {
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		var err error
		userProfile, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("não foi possível localizar o diretório do usuário: %w", err)
		}
	}

	patterns := []string{
		filepath.Join(userProfile, ".vscode", "extensions", "totvs.tds-vscode-*", "node_modules", "@totvs", "tds-ls", "bin", "windows", "advpls.exe"),
		filepath.Join(userProfile, ".vscode", "extensions", "totvs.tds-vscode-*", "node_modules", "@totvs", "tds-ls", "bin", "advpls.exe"),
		filepath.Join(userProfile, "AppData", "Roaming", "npm", "node_modules", "@totvs", "tds-ls", "bin", "windows", "advpls.exe"),
	}

	var tried []string
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err == nil && len(matches) > 0 {
			return matches[len(matches)-1], nil
		}
		tried = append(tried, pattern)
	}

	return "", fmt.Errorf(
		"advpls.exe não encontrado. Verifique se a extensão 'TOTVS Developer Studio' está instalada no VS Code.\n\nCaminhos verificados:\n  - %s",
		strings.Join(tried, "\n  - "),
	)
}
