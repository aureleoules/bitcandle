package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/aureleoules/bitcandle/injection"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/checksum0/go-electrum/electrum"
	"github.com/spf13/cobra"
	"github.com/thediveo/enumflag"
)

var (
	filePath        string
	privateKeyPath  string
	network         Network
	injectionMethod InjectionMethod
	feeRate         int
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

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			errInjectHelp(err.Error())
		}

		fmt.Println(len(data), "bytes loaded.")

		key, err := loadKey(privateKeyPath)
		if err != nil {
			errInjectHelp(err.Error())
		}

		netParams := loadChainParams(network)

		p2shAddr, err := injection.P2SHScriptAddr(data, key.PubKey(), netParams)
		if err != nil {
			errInjectHelp(err.Error())
		}

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

		paymentAmount := len(dummyTx)*feeRate + 546
		fmt.Printf("You must send %.8f BTC to %s.\n", float64(paymentAmount)/100000000, p2shAddr)

		client := electrum.NewServer()
		err = client.ConnectTCP("127.0.0.1:50001")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		script, err := txscript.PayToAddrScript(p2shAddr)
		scriptHash := sha256.Sum256(script)
		reversedScriptHash := ReverseBytes(scriptHash[:])
		reversedScriptHashHex := hex.EncodeToString(reversedScriptHash)

		receivedPayment := false
		var prevOut *wire.OutPoint

		for !receivedPayment {
			history, err := client.GetHistory(reversedScriptHashHex)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}

			for _, h := range history {
				tx, err := client.GetTransaction(h.Hash)
				if err != nil {
					fmt.Println(err)
					os.Exit(1)
				}

				for _, vout := range tx.Vout {
					if vout.ScriptPubkey.Hex == hex.EncodeToString(script) {
						fmt.Println("Payment successfully retrieved.")
						receivedPayment = true
						txHash, err := chainhash.NewHashFromStr(h.Hash)
						if err != nil {
							fmt.Println(err)
							os.Exit(1)
						}
						prevOut = wire.NewOutPoint(txHash, vout.N)
						break
					}
				}
			}

			time.Sleep(time.Second)
		}

		tx, err := injection.P2SHBuildTX(data, prevOut, wire.NewTxOut(546, payToAddrScript), key, netParams)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		txid, err := client.BroadcastTransaction(hex.EncodeToString(tx))
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Successfully injected data on Bitcoin.")
		fmt.Println("File txid:", txid)
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
