package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/btcsuite/btcd/btcec"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(genKeyCmd)
}

var genKeyCmd = &cobra.Command{
	Use:   "genkey",
	Short: "Generate a Bitcoin private key",
	Run: func(cmd *cobra.Command, args []string) {
		var outputPath string
		if len(args) > 0 {
			outputPath = args[0]
		} else {
			outputPath = "key.bin"
		}

		_, err := os.Stat(outputPath)
		if err == nil {
			fmt.Println(outputPath, "already exists.")
			os.Exit(1)
		}

		key, err := btcec.NewPrivateKey(btcec.S256())
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = ioutil.WriteFile(outputPath, key.Serialize(), 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		fmt.Println("Private key successfully generated at \"" + outputPath + "\".")
	},
}

func loadKey(path string) (*btcec.PrivateKey, error) {
	key, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	pKey, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)
	return pKey, nil
}
