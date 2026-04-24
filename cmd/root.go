package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tpm",
	Short: "TOTVS Package Manager",
	Long:  "Gerenciador de dependências de fontes .tlpp/.prw para o ecossistema TOTVS Protheus.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(recompileCmd)
	rootCmd.AddCommand(deleteCmd)
}
