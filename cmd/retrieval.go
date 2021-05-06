package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/aureleoules/bitcandle/retrieval"
	"github.com/checksum0/go-electrum/electrum"
	"github.com/spf13/cobra"
)

var (
	txHash     string
	outputFile string
)

func init() {
	retrieveCmd.Flags().StringVar(&txHash, "tx", "", "txid of the file to retrieve")
	retrieveCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path")

	rootCmd.AddCommand(retrieveCmd)
}

var retrieveCmd = &cobra.Command{
	Use:   "retrieve",
	Short: "Retrieve a file on the Bitcoin network",
	Run: func(cmd *cobra.Command, args []string) {
		if txHash == "" {
			errRetrieveHelp("no txid was provided")
		}

		if outputFile == "" {
			errRetrieveHelp("no output path was specified")
		}

		client := electrum.NewServer()
		err := client.ConnectTCP("127.0.0.1:50001")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		tx, err := client.GetTransaction(txHash)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		data, err := retrieval.P2SHRetrieveData(tx.Vin[0].ScriptSig.Asm)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		err = ioutil.WriteFile(outputFile, data, 0644)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func errRetrieveHelp(err string) {
	fmt.Println("error: " + err)
	fmt.Println(`Please see "bitcandle retrieve --help" for more information.`)
	os.Exit(1)
}
