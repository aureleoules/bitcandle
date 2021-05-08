package cmd

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/aureleoules/bitcandle/electrum"
	"github.com/aureleoules/bitcandle/injector"
	"github.com/briandowns/spinner"
	"github.com/btcsuite/btcd/wire"
	"github.com/guumaster/logsymbols"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

var (
	txHash         string
	outputFile     string
	electrumServer string
)

func init() {
	retrieveCmd.Flags().StringVar(&txHash, "tx", "", "txid of the file to retrieve")
	retrieveCmd.Flags().StringVarP(&outputFile, "output", "o", "", "output file path")
	retrieveCmd.Flags().StringVarP(&electrumServer, "server", "s", "", "electrum server")

	retrieveCmd.PersistentFlags().VarP(
		enumflag.New(&network, "network", NetworkIds, enumflag.EnumCaseInsensitive), "network", "n", "bitcoin network; can be 'mainnet', 'testnet' or 'regtest'")

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

		if electrumServer == "" {
			electrumServer = getDefaultElectrumServer(network)
		}

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithSuffix(" Connecting to electrum server..."))
		s.Start()

		err := electrum.Connect(electrumServer)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not connect to electrum server.")
			os.Exit(1)
		}

		s.Stop()
		fmt.Println(logsymbols.Success, "Connected to electrum server ("+electrumServer+").")

		rawtx, err := electrum.Client.GetRawTransaction(txHash)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not retrieve transaction.")
			os.Exit(1)
		}

		var tx wire.MsgTx
		rawtxBytes, err := hex.DecodeString(rawtx)
		if err != nil {
			s.Stop()
			fmt.Println(logsymbols.Error, "Could not decode transaction hex.")
			os.Exit(1)
		}
		err = tx.Deserialize(bytes.NewReader(rawtxBytes))
		if err != nil {
			s.Stop()
			fmt.Println(logsymbols.Error, "Could not decode transaction.")
			os.Exit(1)
		}

		data, err := injector.P2SHRetrieveData(tx.TxIn)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not parse data.")
			os.Exit(1)
		}

		fmt.Println(logsymbols.Success, "Retrieved file.")

		err = ioutil.WriteFile(outputFile, data, 0644)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not write to output file.")
			os.Exit(1)
		}

		fmt.Println(logsymbols.Success, "Saved file to \""+outputFile+"\".")
	},
}

func errRetrieveHelp(err string) {
	fmt.Println("error: " + err)
	fmt.Println(`Please see "bitcandle retrieve --help" for more information.`)
	os.Exit(1)
}
