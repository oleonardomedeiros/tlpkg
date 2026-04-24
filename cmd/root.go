package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const Version = "0.4.4"

var rootCmd = &cobra.Command{
	Use:     "tlpkg",
	Short:   "TOTVS TLPP Package Manager",
	Long:    "Gerenciador de dependências de fontes .tlpp/.prw para o ecossistema TOTVS Protheus.",
	Version: Version,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate(fmt.Sprintf("tlpkg v%s\n", Version))

	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(recompileCmd)
	rootCmd.AddCommand(deleteCmd)
}
