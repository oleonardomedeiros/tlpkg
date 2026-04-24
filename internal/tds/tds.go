package tds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oleonardomedeiros/tlpkg/internal/config"
	"github.com/oleonardomedeiros/tlpkg/internal/lsp"
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
	return c.runLSP(func(token string, lspClient *lsp.Client) error {
		return lspClient.Compile(token, c.environment, []string{filePath}, c.includes, recompile)
	})
}

func (c *Client) Delete(filePath string) error {
	return c.runLSP(func(token string, lspClient *lsp.Client) error {
		return lspClient.DeletePrograms(token, c.environment, []string{filePath})
	})
}

func (c *Client) runLSP(fn func(token string, client *lsp.Client) error) error {
	lspClient, err := lsp.NewClient(c.advplsPath)
	if err != nil {
		return err
	}
	defer lspClient.Close()

	token, needAuth, err := lspClient.Connect(c.name, c.address, c.port, c.build, c.environment)
	if err != nil {
		return fmt.Errorf("erro ao conectar ao servidor: %w", err)
	}

	if needAuth {
		token, err = lspClient.Authenticate(token, c.environment, "", "")
		if err != nil {
			return fmt.Errorf("autenticação necessária — configure usuário/senha com 'tlpkg config': %w", err)
		}
	}

	defer lspClient.Disconnect(token, c.name)

	return fn(token, lspClient)
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
