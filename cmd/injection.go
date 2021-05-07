package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/aureleoules/bitcandle/consensus"
	"github.com/aureleoules/bitcandle/injection"
	"github.com/briandowns/spinner"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/checksum0/go-electrum/electrum"
	"github.com/guumaster/logsymbols"
	"github.com/mdp/qrterminal"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

var (
	filePath        string
	privateKeyPath  string
	network         Network
	injectionMethod InjectionMethod
	feeRate         int
	changeAddress   string
)

type InjectionMethod enumflag.Flag

// ② Define the enumeration values for FooMode.
const (
	Auto InjectionMethod = iota
	P2FKH
	P2FK
	P2SH
	OP_RETURN
)

var InjectionMethodIds = map[InjectionMethod][]string{
	Auto:      {"auto"},
	P2FKH:     {"p2fkh"},
	P2FK:      {"p2fk"},
	P2SH:      {"p2sh"},
	OP_RETURN: {"op_return"},
}

type Network enumflag.Flag

// ② Define the enumeration values for FooMode.
const (
	Mainnet Network = iota
	Testnet
	RegressionTest
)

var NetworkIds = map[Network][]string{
	Mainnet:        {"mainnet"},
	Testnet:        {"testnet"},
	RegressionTest: {"regtest"},
}

func init() {
	injectCmd.PersistentFlags().VarP(
		enumflag.New(&injectionMethod, "method", InjectionMethodIds, enumflag.EnumCaseInsensitive), "method", "m", "injection method; can be 'auto', 'p2fkh', 'p2fk', 'p2sh' or 'op_return'")

	injectCmd.PersistentFlags().VarP(
		enumflag.New(&network, "network", NetworkIds, enumflag.EnumCaseInsensitive), "network", "n", "bitcoin network; can be 'mainnet', 'testnet' or 'regtest'")

	injectCmd.Flags().StringVarP(&filePath, "file", "f", "", "path of the file to inject on Bitcoin")
	injectCmd.Flags().StringVarP(&privateKeyPath, "key", "k", "key.hex", "path of a private key to sign transactions")
	injectCmd.Flags().StringVarP(&changeAddress, "change-addr", "c", "", "address to receive change (548 sats)")
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

		if privateKeyPath == "" {
			errInjectHelp("missing private key path")
		}

		if electrumServer == "" {
			electrumServer = getDefaultElectrumServer(network)
		}

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			errInjectHelp(err.Error())
		}
		fmt.Println(logsymbols.Success, "Loaded", len(data), "bytes to inject.")

		key, err := loadKey(privateKeyPath)
		if err != nil {
			errInjectHelp(err.Error())
		}
		fmt.Println(logsymbols.Success, "Loaded private key.")

		netParams := loadChainParams(network)

		p2shAddr, err := injection.P2SHScriptAddr(data, key.PubKey(), netParams)
		if err != nil {
			errInjectHelp(err.Error())
		}
		fmt.Println(logsymbols.Success, "Generated P2SH address.")

		addr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(key.PubKey().SerializeCompressed()), netParams)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		payToAddrScript, err := txscript.PayToAddrScript(addr)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		dummyPrevOut := wire.NewOutPoint(chaincfg.MainNetParams.GenesisHash, 0)
		dummyTx, err := injection.P2SHBuildTX(data, dummyPrevOut, wire.NewTxOut(0, payToAddrScript), key, netParams)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		s := spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithSuffix(" Connecting to electrum server..."))
		s.Start()

		client := electrum.NewServer()
		err = client.ConnectTCP(electrumServer)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not connect to electrum server.")
			os.Exit(1)
		}

		s.Stop()
		fmt.Println(logsymbols.Success, "Connected to electrum server ("+electrumServer+").")

		var dummyTxBytes bytes.Buffer
		dummyTx.Serialize(&dummyTxBytes)

		paymentAmount := float64(len(dummyTxBytes.Bytes())*feeRate+consensus.P2PKHDustLimit) / 100_000_000
		fmt.Println(logsymbols.Info, fmt.Sprintf("You must send %.8f BTC to %s.", paymentAmount, p2shAddr))

		qrterminal.GenerateHalfBlock(fmt.Sprintf("bitcoin:%s?amount=%.8f", p2shAddr.EncodeAddress(), paymentAmount), qrterminal.L, os.Stdout)

		script, err := txscript.PayToAddrScript(p2shAddr)
		scriptHash := sha256.Sum256(script)
		reversedScriptHash := ReverseBytes(scriptHash[:])
		reversedScriptHashHex := hex.EncodeToString(reversedScriptHash)

		receivedPayment := false
		var prevOut *wire.OutPoint

		s = spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithSuffix(" Waiting for payment..."))
		s.Start()

		for !receivedPayment {
			history, err := client.GetHistory(reversedScriptHashHex)
			if err != nil {
				s.Stop()
				fmt.Println(logsymbols.Error, "Could not retrieve transactions.")
				os.Exit(1)
			}

			for _, h := range history {
				rawtx, err := client.GetRawTransaction(h.Hash)
				if err != nil {
					s.Stop()
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

				for i, vout := range tx.TxOut {
					if bytes.Equal(vout.PkScript, script) {
						receivedPayment = true
						txHash, err := chainhash.NewHashFromStr(h.Hash)
						if err != nil {
							s.Stop()
							fmt.Println(logsymbols.Error, "Invalid transaction id.")
							os.Exit(1)
						}
						prevOut = wire.NewOutPoint(txHash, uint32(i))
						break
					}
				}
			}
			if !receivedPayment {
				time.Sleep(time.Second)
			}
		}
		s.Stop()
		fmt.Println(logsymbols.Success, "Payment received.")

		tx, err := injection.P2SHBuildTX(data, prevOut, wire.NewTxOut(consensus.P2PKHDustLimit, payToAddrScript), key, netParams)
		if err != nil {
			s.Stop()
			fmt.Println(err)
			os.Exit(1)
		}

		_, err = client.GetRawTransaction(tx.TxHash().String())
		if err == nil {
			fmt.Println(logsymbols.Warn, "Data already injected.")
			fmt.Println(logsymbols.Info, "TxID:", tx.TxHash().String())
		} else {
			s = spinner.New(spinner.CharSets[9], 100*time.Millisecond, spinner.WithSuffix(" Broadcasting transaction..."))
			s.Start()

			var txBytes bytes.Buffer
			tx.Serialize(&txBytes)

			txid, err := client.BroadcastTransaction(hex.EncodeToString(txBytes.Bytes()))
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

func loadChainParams(net Network) *chaincfg.Params {
	switch net {
	case Mainnet:
		return &chaincfg.MainNetParams
	case Testnet:
		return &chaincfg.TestNet3Params
	case RegressionTest:
		return &chaincfg.RegressionNetParams
	}

	return nil
}

func errInjectHelp(err string) {
	fmt.Println("error: " + err)
	fmt.Println(`Please see "bitcandle inject --help" for more information.`)
	os.Exit(1)
}

// ReverseBytes reverses a byte slice
func ReverseBytes(s []byte) []byte {
	res := make([]byte, len(s))
	prevPos, resPos := 0, len(s)
	for pos := range s {
		resPos -= pos - prevPos
		copy(res[resPos:], s[prevPos:pos])
		prevPos = pos
	}
	copy(res[0:], s[prevPos:])
	return res
}
