package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oleonardomedeiros/tlpkg/internal/lockfile"
	"github.com/oleonardomedeiros/tlpkg/internal/parser"
	"github.com/oleonardomedeiros/tlpkg/internal/tds"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove <pacote>",
	Short: "Remove uma dependência: descompila do RPO, deleta o arquivo e remove do packages",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pkgFilePath := filepath.Join(cwd, "packages")
	pkgFile, err := parser.ParsePackagesFile(pkgFilePath)
	if err != nil {
		return err
	}

	// Verifica se o pacote está declarado
	declared := false
	for _, dep := range pkgFile.Dependencies {
		if dep.Name == name {
			declared = true
			break
		}
	}

	if !declared {
		return fmt.Errorf("pacote '%s' não encontrado no arquivo packages", name)
	}

	tdsClient, err := tds.NewClient()
	if err != nil {
		return err
	}

	if !confirmEnvironment(tdsClient.ServerInfo()) {
		fmt.Println("Operação cancelada.")
		return nil
	}

	fmt.Println()

	tlppPath := filepath.Join(cwd, "lib", "packages", name+".tlpp")

	// Descompila do RPO
	if fileExists(tlppPath) {
		fmt.Printf("  descompilando %s do RPO... ", name)
		if err := tdsClient.Delete(tlppPath); err != nil {
			fmt.Println("erro")
			return err
		}
		fmt.Println("ok")

		// Deleta o arquivo
		fmt.Printf("  deletando %s.tlpp... ", name)
		if err := os.Remove(tlppPath); err != nil {
			fmt.Println("erro")
			return fmt.Errorf("erro ao deletar arquivo: %w", err)
		}
		fmt.Println("ok")
	} else {
		fmt.Printf("  arquivo %s.tlpp não encontrado em lib/packages/, pulando descompilação\n", name)
	}

	// Remove do arquivo packages
	fmt.Printf("  removendo do arquivo packages... ")
	if err := removeFromPackagesFile(pkgFilePath, name); err != nil {
		fmt.Println("erro")
		return err
	}
	fmt.Println("ok")

	// Atualiza lock file
	locked, _ := lockfile.Read(cwd)
	delete(locked, name)
	var deps []lockfile.LockedDep
	for k, v := range locked {
		deps = append(deps, lockfile.LockedDep{Name: k, Version: v})
	}
	if err := lockfile.Write(cwd, deps); err != nil {
		return err
	}

	fmt.Printf("\n✓ %s removido.\n", name)
	return nil
}

func removeFromPackagesFile(path, name string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	target := fmt.Sprintf("package '%s'", name)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, target) {
			continue
		}
		lines = append(lines, line)
	}

	// Remove linhas em branco duplicadas no final
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(path, []byte(content), 0644)
}
