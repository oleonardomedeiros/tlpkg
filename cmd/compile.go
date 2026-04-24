package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/oleonardomedeiros/tlpkg/internal/tds"
	"github.com/oleonardomedeiros/tlpkg/internal/vscode"
)

var compileCmd = &cobra.Command{
	Use:   "compile [arquivo]",
	Short: "Compila um arquivo .tlpp/.prw no ambiente ativo",
	Args:  cobra.ExactArgs(1),
	RunE:  runCompile,
}

var recompileCmd = &cobra.Command{
	Use:   "recompile [arquivo]",
	Short: "Recompila forçando reescrita no RPO",
	Args:  cobra.ExactArgs(1),
	RunE:  runRecompile,
}

var deleteCmd = &cobra.Command{
	Use:   "delete [arquivo]",
	Short: "Descompila (remove do RPO) um arquivo",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func runCompile(cmd *cobra.Command, args []string) error {
	client, err := buildTDSClient()
	if err != nil {
		return err
	}

	if !confirmEnvironment(client.ServerInfo()) {
		fmt.Println("Operação cancelada.")
		return nil
	}

	fmt.Printf("Compilando %s...\n", args[0])
	return client.Compile(args[0], false)
}

func runRecompile(cmd *cobra.Command, args []string) error {
	client, err := buildTDSClient()
	if err != nil {
		return err
	}

	if !confirmEnvironment(client.ServerInfo()) {
		fmt.Println("Operação cancelada.")
		return nil
	}

	fmt.Printf("Recompilando %s...\n", args[0])
	return client.Compile(args[0], true)
}

func runDelete(cmd *cobra.Command, args []string) error {
	client, err := buildTDSClient()
	if err != nil {
		return err
	}

	if !confirmEnvironment(client.ServerInfo()) {
		fmt.Println("Operação cancelada.")
		return nil
	}

	fmt.Printf("Descompilando %s...\n", args[0])
	return client.Delete(args[0])
}

func buildTDSClient() (*tds.Client, error) {
	config, err := vscode.LoadServersConfig()
	if err != nil {
		return nil, err
	}

	return tds.NewClient(config)
}

func confirmEnvironment(serverInfo string) bool {
	fmt.Println()
	fmt.Printf("  %s\n", serverInfo)
	fmt.Print("\nConfirmar? [s/N] ")

	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "s" || input == "sim"
}
