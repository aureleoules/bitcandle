package injector

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/aureleoules/bitcandle/consensus"
	"github.com/aureleoules/bitcandle/electrum"
	"github.com/aureleoules/bitcandle/util"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
	"github.com/guumaster/logsymbols"
)

// P2SHInjectionAddress holds informations about a script hash address
// The P2SH address is derived from the user's public key and the file's data
// The user must sends coins to this script hash address.
// The UTXO can be redeemed by providing a signature script containing the corresponding file
type P2SHInjectionAddress struct {
	Address *btcutil.AddressScriptHash
	UTXO    *wire.OutPoint
	Chunks  [][]byte
}

// P2SHInjection holds all necessary information to inject arbitrary data on the Bitcoin network
type P2SHInjection struct {
	Network   *chaincfg.Params
	FeeRate   int
	Addresses []*P2SHInjectionAddress

	parts      [][]byte
	privateKey *btcec.PrivateKey
}

// NewP2SHInjection creates a new data injection structure
func NewP2SHInjection(data []byte, feeRate int, key *btcec.PrivateKey, network *chaincfg.Params) (*P2SHInjection, error) {
	injection := P2SHInjection{
		// Create as many inputs as needed
		// There can only be 1461 bytes encoded in each input
		parts:      dataToChunks(data, consensus.P2SHInputDataLimit),
		Network:    network,
		FeeRate:    feeRate,
		privateKey: key,
		Addresses:  make([]*P2SHInjectionAddress, 0),
	}

	for _, p := range injection.parts {
		// Split data into chunks of 520 bytes
		// This is the maximum of data that can be pushed on the stack at a time
		chunks := dataToChunks(p, consensus.P2SHPushDataLimit)

		// Build redeem script
		redeemScript, err := buildRedeemScript(key.PubKey(), chunks)
		if err != nil {
			return nil, err
		}

		// Hash redeemscript to build the scriptHash address
		addr, err := btcutil.NewAddressScriptHash(redeemScript, network)
		if err != nil {
			return nil, err
		}

		// Insert payment addresses to the structure
		injection.Addresses = append(injection.Addresses, &P2SHInjectionAddress{
			Address: addr,
			Chunks:  chunks,
		})
	}

	return &injection, nil
}

// NumInputs counts the number of inputs required to hold the file
func (i *P2SHInjection) NumInputs() int {
	return len(i.parts)
}

// EstimateCost creates a dummy transaction containing all signature scripts required to store the file
// This allows us to estimate the final transaction size in bytes
func (i *P2SHInjection) EstimateCost() (float64, float64, int, error) {
	// Generate dummy private key
	dummyKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return 0, 0, 0, err
	}

	// Create dummy P2PKH address
	addr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(dummyKey.PubKey().SerializeCompressed()), &chaincfg.RegressionNetParams) // chain params do not matter
	if err != nil {
		return 0, 0, 0, err
	}

	// Build P2PKH dummy script
	payToAddrScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return 0, 0, 0, err
	}

	// Add dummy UTXOs to Tx
	var dummyPrevOuts []*wire.OutPoint
	for k := 0; k < i.NumInputs(); k++ {
		dummyPrevOuts = append(dummyPrevOuts, wire.NewOutPoint(chaincfg.MainNetParams.GenesisHash, 0))
	}

	// Build dummy TX
	dummyTx, err := i.buildTX(wire.NewTxOut(0, payToAddrScript), true)
	if err != nil {
		return 0, 0, 0, err
	}

	var dummyTxBytes bytes.Buffer
	dummyTx.Serialize(&dummyTxBytes)

	// Count tx bytes and estimate cost of transaction
	costSats := float64(len(dummyTxBytes.Bytes())*i.FeeRate + consensus.P2PKHDustLimit)

	// Funds that must be sent to each address
	costPerInput := math.Ceil(costSats/float64(i.NumInputs())) / consensus.BTCSats

	return float64(costSats) / consensus.BTCSats, costPerInput, len(dummyTxBytes.Bytes()), nil
}

// BuildTX constructs the final transaction containing the file
func (i *P2SHInjection) BuildTX(txOut *wire.TxOut) (*wire.MsgTx, error) {
	return i.buildTX(txOut, false)
}

func (i *P2SHInjection) buildTX(txOut *wire.TxOut, dummy bool) (*wire.MsgTx, error) {
	// Create transaction
	tx := wire.NewMsgTx(wire.TxVersion)
	// Add mandatory txout
	tx.AddTxOut(txOut)

	for _, addr := range i.Addresses {
		// Create a dummy UTXO for estimation purposes
		if dummy {
			addr.UTXO = wire.NewOutPoint(chaincfg.MainNetParams.GenesisHash, 0)
		}

		// Add UTXO to the transaction
		txIn := wire.NewTxIn(addr.UTXO, nil, nil)
		tx.AddTxIn(txIn)
	}

	var scriptSigs [][]byte
	for k, addr := range i.Addresses {
		// Sign each input individually
		scriptSig, err := buildSignatureScript(tx, addr.Chunks, i.privateKey, k, dummy)
		if err != nil {
			return nil, err
		}
		// Store script signature separately
		scriptSigs = append(scriptSigs, scriptSig)
	}

	// Once all inputs are signed, add script signatures to their corresponding inputs
	for k := range scriptSigs {
		tx.TxIn[k].SignatureScript = scriptSigs[k]
	}

	return tx, nil
}

func buildSignatureScript(tx *wire.MsgTx, chunks [][]byte, key *btcec.PrivateKey, inputIndex int, dummy bool) ([]byte, error) {
	// The script signature must contain the original redeem script (not hashed)
	redeemScript, err := buildRedeemScript(key.PubKey(), chunks)
	if err != nil {
		return nil, err
	}

	var sig []byte
	if dummy {
		// Empty signature of max possible size
		sig = make([]byte, consensus.ECDSAMaxSignatureSize)
	} else {
		// Sign transaction pre-image
		sig, err = txscript.RawTxInSignature(tx, inputIndex, redeemScript, txscript.SigHashAll, key)
		if err != nil {
			return nil, err
		}
	}

	// Create input script
	inputScript := txscript.NewScriptBuilder()

	// Push raw signature
	inputScript.AddData(sig)

	// Push each chunk of data (520 bytes max)
	for _, chunk := range chunks {
		inputScript.AddData(chunk)
	}

	// Push original redeem script on the top of the stack
	inputScript.AddData(redeemScript)

	// Return serialized P2SH input script
	return inputScript.Script()
}

func buildRedeemScript(pubKey *btcec.PublicKey, chunks [][]byte) ([]byte, error) {
	redeemScript := txscript.NewScriptBuilder()

	// Reverse traversal of chunks such that the stack is popped in the correct order
	for i := len(chunks) - 1; i >= 0; i-- {
		// Hash each chunk of data such that chunks cannot be ordered differently by tx relay nodes or miners
		// This ensures integrity of the data
		redeemScript.AddOp(txscript.OP_HASH160)
		redeemScript.AddData(btcutil.Hash160(chunks[i]))
		redeemScript.AddOp(txscript.OP_EQUALVERIFY)
	}

	// Verify tx signature such that the transaction output cannot be redirected to another address
	// This may not be useful if vout value is equal or close to a dust amount as removing the signature verification would save at most 107 bytes (73 sig + 33 pub + 1 opcode)
	redeemScript.AddData(pubKey.SerializeCompressed())
	redeemScript.AddOp(txscript.OP_CHECKSIG)

	// Return serialized P2SH redeem script
	return redeemScript.Script()
}

// WaitPayments waits until all required UTXOs are created on all pre-generated P2SH addresses
func (i *P2SHInjection) WaitPayments(onPayment func(addr string, num int)) error {
	var wg sync.WaitGroup
	// Add number of utxos to wait for
	wg.Add(i.NumInputs())

	_, costPerInput, _, err := i.EstimateCost()
	if err != nil {
		return err
	}

	// Count the number of UTXOs received
	var paymentsReceived int

	for j, p2shAddr := range i.Addresses {
		go func(addr *btcutil.AddressScriptHash, j int) {
			// Mark job as done
			defer wg.Done()

			for {
				script, err := txscript.PayToAddrScript(addr)
				if err != nil {
					fmt.Println(logsymbols.Error, "Could not build P2SH script.")
				}
				// Electrum servers only accept the hash of the scriptPubKey (in reverse)
				scriptHash := sha256.Sum256(script)
				reversedScriptHash := util.ReverseBytes(scriptHash[:])
				reversedScriptHashHex := hex.EncodeToString(reversedScriptHash)

				// Check all received transactions of a P2SH address
				history, err := electrum.Client.GetHistory(reversedScriptHashHex)
				if err != nil {
					return
				}

				for _, h := range history {
					// Some electrum servers do not support GetTransaction, so we have to decode the raw transaction manually
					rawtx, err := electrum.Client.GetRawTransaction(h.Hash)
					if err != nil {
						fmt.Println(logsymbols.Error, "Could not retrieve transaction.")
						continue
					}

					// Decode raw transaction
					var tx wire.MsgTx
					rawtxBytes, err := hex.DecodeString(rawtx)
					if err != nil {
						return
					}
					err = tx.Deserialize(bytes.NewReader(rawtxBytes))
					if err != nil {
						return
					}

					for k, vout := range tx.TxOut {
						// Check that the payment address corresponds to the P2SH address and that enough bitcoins were sent
						if bytes.Equal(vout.PkScript, script) && vout.Value >= int64(costPerInput*consensus.BTCSats) {
							txHash, err := chainhash.NewHashFromStr(h.Hash)
							if err != nil {
								return
							}
							// Add utxo to corresponding P2SH address
							i.Addresses[j].UTXO = wire.NewOutPoint(txHash, uint32(k))
							paymentsReceived++

							// Event
							onPayment(addr.EncodeAddress(), paymentsReceived)
							return
						}
					}
				}

				// Check for new payments every second
				time.Sleep(time.Second)
			}
		}(p2shAddr.Address, j)
	}

	// Wait until all payments are received
	wg.Wait()
	return nil
}
