package tds

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/oleonardomedeiros/tpm/internal/vscode"
)

type Client struct {
	server   *vscode.Server
	includes []string
	advplsPath string
}

func NewClient(config *vscode.ServersConfig) (*Client, error) {
	server, err := vscode.ActiveServer(config)
	if err != nil {
		return nil, err
	}

	advplsPath, err := findAdvpls()
	if err != nil {
		return nil, err
	}

	return &Client{
		server:     server,
		includes:   config.Includes,
		advplsPath: advplsPath,
	}, nil
}

func (c *Client) ServerInfo() string {
	return fmt.Sprintf("servidor: %s (%s:%d) | ambiente: %s",
		c.server.Name,
		c.server.Address,
		c.server.Port,
		c.server.CurrentEnvironment,
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
		action,
		fmt.Sprintf("serverType=AdvPL"),
		fmt.Sprintf("server=%s", c.server.Address),
		fmt.Sprintf("port=%d", c.server.Port),
		fmt.Sprintf("build=%s", c.server.BuildVersion),
		fmt.Sprintf("environment=%s", c.server.CurrentEnvironment),
		fmt.Sprintf("includes=%s", strings.Join(c.includes, ";")),
		fmt.Sprintf("program=%s", filePath),
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
	// Procura advpls.exe dentro das extensões do VS Code
	appData := os.Getenv("APPDATA")
	if appData == "" {
		home, _ := os.UserHomeDir()
		appData = filepath.Join(home, "AppData", "Roaming")
	}

	// Substituindo APPDATA por USERPROFILE para extensões do VS Code
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		var err error
		userProfile, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("não foi possível localizar o diretório do usuário: %w", err)
		}
	}

	pattern := filepath.Join(userProfile, ".vscode", "extensions", "totvs.tds-vscode-*", "node_modules", "@totvs", "tds-ls", "bin", "windows", "advpls.exe")
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("advpls.exe não encontrado\ncertifique-se que a extensão TOTVS Developer Studio está instalada no VS Code")
	}

	// Usa a versão mais recente encontrada
	return matches[len(matches)-1], nil
}
