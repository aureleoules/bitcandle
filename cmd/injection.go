package cmd

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/aureleoules/bitcandle/consensus"
	"github.com/aureleoules/bitcandle/electrum"
	"github.com/aureleoules/bitcandle/injector"
	"github.com/briandowns/spinner"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/guumaster/logsymbols"
	"github.com/mdp/qrterminal"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

const keyDir = "./keys"

var (
	filePath      string
	network       Network
	feeRate       int
	changeAddress string
)

// Network represents an enum of different bitcoin networks
type Network enumflag.Flag

// ② Define the enumeration values for FooMode.
const (
	Mainnet Network = iota
	Testnet
	RegressionTest
)

// NetworkIds mapper
var NetworkIds = map[Network][]string{
	Mainnet:        {"mainnet"},
	Testnet:        {"testnet"},
	RegressionTest: {"regtest"},
}

func init() {
	injectCmd.PersistentFlags().VarP(
		enumflag.New(&network, "network", NetworkIds, enumflag.EnumCaseInsensitive), "network", "n", "bitcoin network; can be 'mainnet', 'testnet' or 'regtest'")

	injectCmd.Flags().StringVarP(&filePath, "file", "f", "", "path of the file to inject on Bitcoin")
	injectCmd.Flags().StringVarP(&changeAddress, "change-address", "c", "", "address to receive change (548 sats)")
	injectCmd.Flags().IntVar(&feeRate, "fee", 5, "fee rate (sat/B)")

	rootCmd.AddCommand(injectCmd)
}

var injectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a file on the Bitcoin network",
	Run: func(cmd *cobra.Command, args []string) {
		if filePath == "" {
			errInjectHelp("missing file path")
		}

		if electrumServer == "" {
			electrumServer = getDefaultElectrumServer(network)
		}

		fileInfo, err := os.Stat(filePath)
		if err != nil {
			errInjectHelp(err.Error())
		}

		// Load file to inject
		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			errInjectHelp(err.Error())
		}

		fmt.Println(logsymbols.Success, "Loaded", len(data), "bytes to inject.")

		if len(data) < 800 {
			fmt.Println(logsymbols.Warn, "It is not recommended to use this injection method for files less than 800 bytes as there are more optimized ones for smaller files.")
		}

		if changeAddress == "" {
			fmt.Println(logsymbols.Warn, "No change address has been provided. Defaulting to provided public key's P2PKH address.")
		}

		if len(data) > consensus.P2SHInputDataLimit {
			fmt.Println(logsymbols.Warn, fmt.Sprintf("File is too large (> %d bytes) for a single input.", consensus.P2SHInputDataLimit))
		}

		md5hasher := md5.New()
		md5hasher.Write(data)
		md5Hash := md5hasher.Sum(nil)

		_ = os.Mkdir(keyDir, 0777)
		keyFilePath := keyDir + "/" + fileInfo.Name() + "_" + hex.EncodeToString(md5Hash)
		_, err = os.Stat(keyFilePath)
		var key *btcec.PrivateKey
		if err == nil {
			key, err = loadKey(keyFilePath)
			if err != nil {
				errInjectHelp(err.Error())
			}

			// Load private key
			fmt.Println(logsymbols.Success, "Loaded existing private key.")
		} else {

			key, err = btcec.NewPrivateKey(btcec.S256())
			if err != nil {
				errInjectHelp(err.Error())
			}

			err = ioutil.WriteFile(keyFilePath, key.Serialize(), 0644)
			if err != nil {
				fmt.Println("here")
				fmt.Println(err)
				os.Exit(1)
			}

			// Load private key
			fmt.Println(logsymbols.Success, "Generated new private key.")
		}

		// Load chain params
		netParams := loadChainParams(network)

		var addr btcutil.Address

		if changeAddress != "" {
			addr, err = btcutil.DecodeAddress(changeAddress, netParams)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		} else {
			addr, err = btcutil.NewAddressPubKeyHash(btcutil.Hash160(key.PubKey().SerializeCompressed()), netParams)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
		}

		// Create file injector
		inject, err := injector.NewP2SHInjection(data, feeRate, key, netParams)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not prepare injection data.")
			os.Exit(1)
		}

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithSuffix(" Connecting to electrum server..."))
		s.Start()

		// Connect to electrum
		err = electrum.Connect(electrumServer)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not connect to electrum server.")
			os.Exit(1)
		}

		s.Stop()
		fmt.Println(logsymbols.Success, "Connected to electrum server ("+electrumServer+").")

		cost, costPerInput, txBytesLen, err := inject.EstimateCost()
		if err != nil {
			fmt.Println(err)
			fmt.Println(logsymbols.Error, "Could not estimate injection cost.")
			os.Exit(1)
		}

		fmt.Println(txBytesLen)
		if txBytesLen > 101000 {
			fmt.Println(logsymbols.Error, "File is too large.")
			os.Exit(1)
		}

		fmt.Println(logsymbols.Info, fmt.Sprintf("Estimated injection cost: %.8f BTC.", cost))

		for _, addr := range inject.Addresses {
			fmt.Println(logsymbols.Info, fmt.Sprintf("You must send %.8f BTC to %s.", costPerInput, addr.Address.EncodeAddress()))

			if len(inject.Addresses) == 1 {
				qrterminal.GenerateHalfBlock(fmt.Sprintf("bitcoin:%s?amount=%.8f", addr.Address.EncodeAddress(), costPerInput), qrterminal.L, os.Stdout)
			}
		}

		if len(inject.Addresses) > 1 {
			fmt.Println(logsymbols.Info, "Copy paste this in Electrum -> Tools -> Pay to many.")
			fmt.Println()
			for _, addr := range inject.Addresses {
				fmt.Println(fmt.Sprintf("%s,%.8f", addr.Address.EncodeAddress(), costPerInput))
			}
			fmt.Println()
		}

		s = spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithSuffix(" Waiting for payments..."))
		s.Start()

		// Wait for utxos to be created by the user
		err = inject.WaitPayments(func(addr string, num int) {
			s.Stop()
			fmt.Println(logsymbols.Success, fmt.Sprintf("Payment received. (%d/%d)", num, inject.NumInputs()))
			s.Start()
		})

		if err != nil {
			fmt.Println(logsymbols.Error, err.Error())
			s.Stop()
			os.Exit(1)
		}

		fmt.Println(logsymbols.Success, "All payments received.")

		payToAddrScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		tx, err := inject.BuildTX(wire.NewTxOut(consensus.P2PKHDustLimit, payToAddrScript))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Checks if transaction has been mined already
		_, err = electrum.Client.GetRawTransaction(tx.TxHash().String())
		if err == nil {
			fmt.Println(logsymbols.Warn, "Data already injected.")
			fmt.Println(logsymbols.Info, "TxID:", tx.TxHash().String())
		} else {
			s = spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithSuffix(" Broadcasting transaction..."))
			s.Start()

			var txBytes bytes.Buffer
			tx.Serialize(&txBytes)

			txid, err := electrum.Client.BroadcastTransaction(hex.EncodeToString(txBytes.Bytes()))
			if err != nil {
				s.Stop()
				fmt.Println(logsymbols.Error, "Could not broadcast transaction.")
				fmt.Println(err)
				os.Exit(1)
			}
			s.Stop()
			fmt.Println(logsymbols.Success, "Data injected.")
			fmt.Println(logsymbols.Info, "TxID:", txid)
		}
	},
}

func errInjectHelp(err string) {
	fmt.Println("error: " + err)
	fmt.Println(`Please see "bitcandle inject --help" for more information.`)
	os.Exit(1)
}
