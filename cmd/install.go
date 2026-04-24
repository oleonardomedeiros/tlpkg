package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/oleonardomedeiros/tpm/internal/lockfile"
	"github.com/oleonardomedeiros/tpm/internal/parser"
	"github.com/oleonardomedeiros/tpm/internal/registry"
	"github.com/oleonardomedeiros/tpm/internal/tds"
	"github.com/oleonardomedeiros/tpm/internal/vscode"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Instala as dependências declaradas no arquivo packages",
	RunE:  runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pkgFile, err := parser.ParsePackagesFile(filepath.Join(cwd, "packages"))
	if err != nil {
		return err
	}

	if len(pkgFile.Dependencies) == 0 {
		fmt.Println("Nenhuma dependência declarada no arquivo packages.")
		return nil
	}

	vsConfig, err := vscode.LoadServersConfig()
	if err != nil {
		return err
	}

	tdsClient, err := tds.NewClient(vsConfig)
	if err != nil {
		return err
	}

	if !confirmEnvironment(tdsClient.ServerInfo()) {
		fmt.Println("Operação cancelada.")
		return nil
	}

	fmt.Println()

	regClient := registry.NewClient(pkgFile.Source)

	fmt.Print("Buscando índice do registry... ")
	index, err := regClient.FetchIndex()
	if err != nil {
		return err
	}
	fmt.Println("ok")

	locked, err := lockfile.Read(cwd)
	if err != nil {
		return err
	}

	libDir := filepath.Join(cwd, "lib", "packages")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return err
	}

	// Resolve e baixa dependências novas ou desatualizadas
	var resolvedDeps []lockfile.LockedDep
	declaredNames := make(map[string]bool)

	for _, dep := range pkgFile.Dependencies {
		declaredNames[dep.Name] = true

		resolved, err := regClient.ResolveVersion(index, dep.Name, dep.Version)
		if err != nil {
			return err
		}

		resolvedDeps = append(resolvedDeps, lockfile.LockedDep{Name: dep.Name, Version: resolved})

		if locked[dep.Name] == resolved {
			if fileExists(filepath.Join(libDir, dep.Name+".tlpp")) {
				fmt.Printf("  ✓ %s %s (já instalado)\n", dep.Name, resolved)
				continue
			}
		}

		fmt.Printf("  ↓ %s %s... ", dep.Name, resolved)
		if _, err := regClient.Download(dep.Name, resolved, libDir); err != nil {
			fmt.Println("erro")
			return err
		}
		fmt.Println("ok")
	}

	// Identifica e remove órfãos
	orphans, err := findOrphans(libDir, declaredNames)
	if err != nil {
		return err
	}

	if len(orphans) > 0 {
		fmt.Println()
		fmt.Println("Removendo dependências não declaradas:")
		for _, orphan := range orphans {
			fmt.Printf("  ✗ %s\n", filepath.Base(orphan))

			fmt.Printf("    descompilando... ")
			if err := tdsClient.Delete(orphan); err != nil {
				fmt.Println("erro")
				return err
			}
			fmt.Println("ok")

			if err := os.Remove(orphan); err != nil {
				return fmt.Errorf("erro ao deletar %s: %w", orphan, err)
			}
		}
	}

	// Compila tudo que está em lib/packages/
	fmt.Println()
	fmt.Println("Compilando dependências:")
	files, err := filepath.Glob(filepath.Join(libDir, "*.tlpp"))
	if err != nil {
		return err
	}

	for _, f := range files {
		fmt.Printf("  ⚙ %s... ", filepath.Base(f))
		if err := tdsClient.Compile(f, false); err != nil {
			fmt.Println("erro")
			return err
		}
		fmt.Println("ok")
	}

	// Salva o lock file
	if err := lockfile.Write(cwd, resolvedDeps); err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("✓ %d dependência(s) instalada(s). libs.lock atualizado.\n", len(resolvedDeps))
	return nil
}

func findOrphans(libDir string, declared map[string]bool) ([]string, error) {
	files, err := filepath.Glob(filepath.Join(libDir, "*.tlpp"))
	if err != nil {
		return nil, err
	}

	var orphans []string
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".tlpp")
		if !declared[name] {
			orphans = append(orphans, f)
		}
	}
	return orphans, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
