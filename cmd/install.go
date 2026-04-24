package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oleonardomedeiros/tlpkg/internal/lockfile"
	"github.com/oleonardomedeiros/tlpkg/internal/parser"
	"github.com/oleonardomedeiros/tlpkg/internal/registry"
	"github.com/oleonardomedeiros/tlpkg/internal/tds"
	"github.com/oleonardomedeiros/tlpkg/internal/vscode"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [pacote] [versão]",
	Short: "Instala dependências. Sem argumentos instala tudo do arquivo packages",
	Example: `  tpm install                        # instala todas as dependências
  tpm install api-financeiro          # instala a versão mais recente
  tpm install api-financeiro 1.2.0    # instala versão específica`,
	Args: cobra.MaximumNArgs(2),
	RunE: runInstall,
}

func init() {
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	pkgFilePath := filepath.Join(cwd, "packages")
	pkgFile, err := parser.ParsePackagesFile(pkgFilePath)
	if err != nil {
		return err
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
	fmt.Println()

	libDir := filepath.Join(cwd, "lib", "packages")
	if err := os.MkdirAll(libDir, 0755); err != nil {
		return err
	}

	// tpm install <nome> [versão] — adiciona ao packages e instala
	if len(args) > 0 {
		return installSingle(cwd, pkgFilePath, pkgFile, regClient, index, tdsClient, libDir, args)
	}

	// tpm install — instala tudo do packages file
	return installAll(cwd, pkgFile, regClient, index, tdsClient, libDir)
}

func installSingle(cwd, pkgFilePath string, pkgFile *parser.PackagesFile, regClient *registry.Client, index *registry.Index, tdsClient *tds.Client, libDir string, args []string) error {
	name := args[0]
	version := "latest"
	if len(args) == 2 {
		version = args[1]
	}

	resolved, err := regClient.ResolveVersion(index, name, version)
	if err != nil {
		return err
	}

	// Adiciona ao arquivo packages se ainda não estiver
	alreadyDeclared := false
	for _, dep := range pkgFile.Dependencies {
		if dep.Name == name {
			alreadyDeclared = true
			break
		}
	}

	if !alreadyDeclared {
		line := fmt.Sprintf("\npackage '%s', '%s'\n", name, resolved)
		f, err := os.OpenFile(pkgFilePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("erro ao atualizar arquivo packages: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(line); err != nil {
			return fmt.Errorf("erro ao escrever no arquivo packages: %w", err)
		}
		fmt.Printf("adicionado ao packages: %s %s\n\n", name, resolved)
	}

	fmt.Printf("  ↓ %s %s... ", name, resolved)
	if _, err := regClient.Download(name, resolved, libDir); err != nil {
		fmt.Println("erro")
		return err
	}
	fmt.Println("ok")

	fmt.Printf("  ⚙ compilando... ")
	if err := tdsClient.Compile(filepath.Join(libDir, name+".tlpp"), false); err != nil {
		fmt.Println("erro")
		return err
	}
	fmt.Println("ok")

	// Atualiza lock
	locked, _ := lockfile.Read(cwd)
	locked[name] = resolved
	var deps []lockfile.LockedDep
	for k, v := range locked {
		deps = append(deps, lockfile.LockedDep{Name: k, Version: v})
	}
	if err := lockfile.Write(cwd, deps); err != nil {
		return err
	}

	fmt.Printf("\n✓ %s %s instalado.\n", name, resolved)
	return nil
}

func installAll(cwd string, pkgFile *parser.PackagesFile, regClient *registry.Client, index *registry.Index, tdsClient *tds.Client, libDir string) error {
	if len(pkgFile.Dependencies) == 0 {
		fmt.Println("Nenhuma dependência declarada no arquivo packages.")
		return nil
	}

	locked, err := lockfile.Read(cwd)
	if err != nil {
		return err
	}

	var resolvedDeps []lockfile.LockedDep
	declaredNames := make(map[string]bool)

	for _, dep := range pkgFile.Dependencies {
		declaredNames[dep.Name] = true

		resolved, err := regClient.ResolveVersion(index, dep.Name, dep.Version)
		if err != nil {
			return err
		}

		resolvedDeps = append(resolvedDeps, lockfile.LockedDep{Name: dep.Name, Version: resolved})

		if locked[dep.Name] == resolved && fileExists(filepath.Join(libDir, dep.Name+".tlpp")) {
			fmt.Printf("  ✓ %s %s (já instalado)\n", dep.Name, resolved)
			continue
		}

		fmt.Printf("  ↓ %s %s... ", dep.Name, resolved)
		if _, err := regClient.Download(dep.Name, resolved, libDir); err != nil {
			fmt.Println("erro")
			return err
		}
		fmt.Println("ok")
	}

	// Remove órfãos
	orphans, err := findOrphans(libDir, declaredNames)
	if err != nil {
		return err
	}

	if len(orphans) > 0 {
		fmt.Println()
		fmt.Println("Removendo dependências não declaradas:")
		for _, orphan := range orphans {
			fmt.Printf("  ✗ %s... ", filepath.Base(orphan))
			if err := tdsClient.Delete(orphan); err != nil {
				fmt.Println("erro")
				return err
			}
			if err := os.Remove(orphan); err != nil {
				return fmt.Errorf("erro ao deletar %s: %w", orphan, err)
			}
			fmt.Println("ok")
		}
	}

	// Compila tudo
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

	if err := lockfile.Write(cwd, resolvedDeps); err != nil {
		return err
	}

	fmt.Printf("\n✓ %d dependência(s) instalada(s). libs.lock atualizado.\n", len(resolvedDeps))
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
