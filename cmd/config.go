package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/oleonardomedeiros/tlpkg/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configura a conexão com o AppServer manualmente",
	Long: `Configura servidor, porta, ambiente e includes do AppServer.
Use quando a extensão TOTVS do VS Code não for detectada automaticamente.`,
	Example: `  tlpkg config
  tlpkg config --server localhost --port 1234 --env P12 --build 7.00.170117A`,
	RunE: runConfig,
}

var (
	flagServer   string
	flagPort     int
	flagEnv      string
	flagBuild    string
	flagIncludes string
)

func init() {
	configCmd.Flags().StringVar(&flagServer, "server", "", "Endereço do AppServer")
	configCmd.Flags().IntVar(&flagPort, "port", 0, "Porta do AppServer")
	configCmd.Flags().StringVar(&flagEnv, "env", "", "Nome do ambiente")
	configCmd.Flags().StringVar(&flagBuild, "build", "", "Versão do build (ex: 7.00.170117A)")
	configCmd.Flags().StringVar(&flagIncludes, "includes", "", "Pastas de includes separadas por ;")
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	// Se passou flags, salva direto
	if flagServer != "" || flagPort != 0 || flagEnv != "" || flagBuild != "" {
		return saveFromFlags()
	}

	// Senão, wizard interativo
	return runWizard()
}

func saveFromFlags() error {
	cfg := &config.TlpkgConfig{
		Server:      flagServer,
		Port:        flagPort,
		Environment: flagEnv,
		Build:       flagBuild,
	}

	if flagIncludes != "" {
		cfg.Includes = strings.Split(flagIncludes, ";")
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	path, _ := config.ConfigPath()
	fmt.Printf("✓ Configuração salva em %s\n", path)
	return nil
}

func runWizard() error {
	reader := bufio.NewReader(os.Stdin)

	existing, _ := config.Load()

	fmt.Println("Configuração do tlpkg")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("Pressione Enter para manter o valor atual.")
	fmt.Println()

	cfg := &config.TlpkgConfig{}
	if existing != nil {
		*cfg = *existing
	}

	cfg.Server = prompt(reader, "Servidor", cfg.Server)
	cfg.Port = promptInt(reader, "Porta", cfg.Port)
	cfg.Environment = prompt(reader, "Ambiente", cfg.Environment)
	cfg.Build = prompt(reader, "Build (ex: 7.00.170117A)", cfg.Build)

	includesStr := strings.Join(cfg.Includes, ";")
	includesInput := prompt(reader, "Includes (separados por ;)", includesStr)
	if includesInput != "" {
		cfg.Includes = strings.Split(includesInput, ";")
	}

	fmt.Println()
	fmt.Printf("  Servidor:  %s:%d\n", cfg.Server, cfg.Port)
	fmt.Printf("  Ambiente:  %s\n", cfg.Environment)
	fmt.Printf("  Build:     %s\n", cfg.Build)
	fmt.Printf("  Includes:  %s\n", strings.Join(cfg.Includes, "; "))
	fmt.Println()
	fmt.Print("Salvar? [S/n] ")

	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer == "n" || answer == "nao" || answer == "não" {
		fmt.Println("Configuração cancelada.")
		return nil
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	path, _ := config.ConfigPath()
	fmt.Printf("\n✓ Configuração salva em %s\n", path)
	return nil
}

func prompt(reader *bufio.Reader, label, current string) string {
	if current != "" {
		fmt.Printf("%s [%s]: ", label, current)
	} else {
		fmt.Printf("%s: ", label)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return current
	}
	return input
}

func promptInt(reader *bufio.Reader, label string, current int) int {
	defaultStr := ""
	if current != 0 {
		defaultStr = strconv.Itoa(current)
	}

	result := prompt(reader, label, defaultStr)
	if result == "" {
		return current
	}

	n, err := strconv.Atoi(result)
	if err != nil {
		return current
	}
	return n
}
