package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Inicializa um projeto TPM no diretório atual",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("erro ao obter diretório atual: %w", err)
	}

	if err := createPackagesFile(cwd); err != nil {
		return err
	}

	if err := createLibsDir(cwd); err != nil {
		return err
	}

	fmt.Println("Projeto TPM inicializado.")
	fmt.Println()
	fmt.Println("  packages       ← declare suas dependências aqui")
	fmt.Println("  libs.lock      ← gerado automaticamente pelo tlpkg install")
	fmt.Println("  lib/packages/  ← arquivos .tlpp baixados")

	return nil
}

func createPackagesFile(dir string) error {
	path := filepath.Join(dir, "packages")

	if _, err := os.Stat(path); err == nil {
		fmt.Println("packages já existe, mantendo arquivo.")
		return nil
	}

	content := "source 'https://cdn.jsdelivr.net/gh/oleonardomedeiros/tpm-registry@main'\n\n# Dependências do projeto\n# package 'api-financeiro', '1.2.0'\n# package 'lib-fiscal', '^2.0.0'\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("erro ao criar arquivo packages: %w", err)
	}

	fmt.Println("criado: packages")
	return nil
}

func createLibsDir(dir string) error {
	libPath := filepath.Join(dir, "lib", "packages")

	if err := os.MkdirAll(libPath, 0755); err != nil {
		return fmt.Errorf("erro ao criar lib/packages/: %w", err)
	}

	fmt.Println("criado: lib/packages/")
	return nil
}
