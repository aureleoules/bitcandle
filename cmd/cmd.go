package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {}

var rootCmd = &cobra.Command{
	Use: "bitcandle",
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// Run CMD
func Run() {
	if err := rootCmd.Execute(); err != nil {
		_ = rootCmd.Help()
		os.Exit(1)
	}
}

func getDefaultElectrumServer(network Network) string {
	switch network {
	case Mainnet:
		return "blockstream.info:110"
	case Testnet:
		return "blockstream.info:143"
	case RegressionTest:
		return "localhost:50001"
	}
	return ""
}
