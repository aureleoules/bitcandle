package injector

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
)

// InjectionAddress holds informations about a script hash address
// The P2SH-P2WSH address is derived from the user's public key and the file's data
// The user must sends coins to this script hash address.
// The UTXO can be redeemed by providing a signature script containing the corresponding file
type InjectionAddress struct {
	Address *btcutil.AddressScriptHash
	UTXO    *wire.OutPoint
	Amount  int64
	Chunks  [][]byte
}

// Injection holds all necessary information to inject arbitrary data on the Bitcoin network
type Injection struct {
	Network   *chaincfg.Params
	FeeRate   int
	Addresses []*InjectionAddress

	parts      [][]byte
	privateKey *btcec.PrivateKey
}

// NewInjection creates a new data injection structure
func NewInjection(data []byte, feeRate int, key *btcec.PrivateKey, network *chaincfg.Params) (*Injection, error) {
	injection := Injection{
		// Create as many inputs as needed
		parts:      dataToChunks(data, (consensus.P2SHP2WSHStackItems-1)*consensus.P2SHP2WSHPushDataLimit),
		Network:    network,
		FeeRate:    feeRate,
		privateKey: key,
		Addresses:  make([]*InjectionAddress, 0),
	}

	for _, p := range injection.parts {
		// Split data into chunks of 520 bytes
		// This is the maximum of data that can be pushed on the stack at a time
		chunks := dataToChunks(p, consensus.P2SHP2WSHPushDataLimit)

		// Build witness script
		witnessScript, err := buildWitnessScript(key.PubKey(), chunks)
		if err != nil {
			return nil, err
		}

		// Build redeem script (OP_0 0x20 [witness prog])
		redeemScript, err := buildRedeemScript(buildWitnessProg(witnessScript))
		if err != nil {
			return nil, err
		}

		// Hash redeemscript to build the scriptHash address
		addr, err := btcutil.NewAddressScriptHash(redeemScript, network)
		if err != nil {
			return nil, err
		}

		// Insert payment addresses to the structure
		injection.Addresses = append(injection.Addresses, &InjectionAddress{
			Address: addr,
			Chunks:  chunks,
		})

		_, amount, err := injection.EstimateCost()
		if err != nil {
			return nil, err
		}

		// Set required UTXO amount for each address
		for i := range injection.Addresses {
			injection.Addresses[i].Amount = amount
		}
	}

	return &injection, nil
}

// NumInputs counts the number of inputs required to store the file
func (i *Injection) NumInputs() int {
	return len(i.parts)
}

// EstimateCost creates a dummy transaction containing all signature scripts required to store the file
// This allows us to estimate the final transaction size in bytes
func (i *Injection) EstimateCost() (int64, int64, error) {
	// Generate dummy private key
	dummyKey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return 0, 0, err
	}

	// Create dummy P2PKH address
	addr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(dummyKey.PubKey().SerializeCompressed()), &chaincfg.RegressionNetParams) // chain params do not matter
	if err != nil {
		return 0, 0, err
	}

	// Build P2PKH dummy script
	payToAddrScript, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return 0, 0, err
	}

	// Add dummy UTXOs to Tx
	var dummyPrevOuts []*wire.OutPoint
	for k := 0; k < i.NumInputs(); k++ {
		dummyPrevOuts = append(dummyPrevOuts, wire.NewOutPoint(chaincfg.MainNetParams.GenesisHash, 0))
	}

	// Build dummy TX
	dummyTx, err := i.buildTX(wire.NewTxOut(0, payToAddrScript), true)
	if err != nil {
		return 0, 0, err
	}

	var dummyTxBytes bytes.Buffer
	var dummyTxNoWitBytes bytes.Buffer

	dummyTx.SerializeNoWitness(&dummyTxNoWitBytes)
	dummyTx.Serialize(&dummyTxBytes)

	// Segwit tx size
	virtualSize := ((3*len(dummyTxNoWitBytes.Bytes()) + len(dummyTxBytes.Bytes())) + 3) / 4
	// Count tx bytes and estimate cost of transaction
	costSats := virtualSize*i.FeeRate + consensus.P2PKHDustLimit

	// Funds that must be sent to each address
	costPerInput := (costSats + i.NumInputs() - 1) / i.NumInputs()
	return int64(costSats), int64(costPerInput), nil
}

// BuildTX constructs the final transaction containing the file
func (i *Injection) BuildTX(txOut *wire.TxOut) (*wire.MsgTx, error) {
	return i.buildTX(txOut, false)
}

func (i *Injection) buildTX(txOut *wire.TxOut, dummy bool) (*wire.MsgTx, error) {
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

	var witnesses []wire.TxWitness
	for k, addr := range i.Addresses {
		// Sign each input individually
		witness, err := buildWitness(tx, addr.Chunks, i.privateKey, k, addr.Amount, dummy)
		if err != nil {
			return nil, err
		}
		// Store script signature separately
		witnesses = append(witnesses, witness)
	}

	// Once all inputs are signed, add script signatures to their corresponding inputs
	for k := range witnesses {
		tx.TxIn[k].Witness = witnesses[k]

		// Build witness script
		witnessScript, err := buildWitnessScript(i.privateKey.PubKey(), i.Addresses[k].Chunks)
		if err != nil {
			return nil, err
		}

		// Push redeem script only
		s := txscript.NewScriptBuilder()
		redeemScript, _ := buildRedeemScript(buildWitnessProg(witnessScript))
		s.AddData(redeemScript)

		tx.TxIn[k].SignatureScript, err = s.Script()
		if err != nil {
			return nil, err
		}
	}

	return tx, nil
}

func buildWitness(tx *wire.MsgTx, chunks [][]byte, key *btcec.PrivateKey, inputIndex int, inputAmount int64, dummy bool) ([][]byte, error) {
	// The script signature must contain the original redeem script (not hashed)
	witnessScript, err := buildWitnessScript(key.PubKey(), chunks)
	if err != nil {
		return nil, err
	}

	var sig []byte
	if dummy {
		// Empty signature of max possible size
		sig = make([]byte, consensus.ECDSAMaxSignatureSize)
	} else {
		sigHash := txscript.NewTxSigHashes(tx)

		// Sign transaction pre-image
		sig, err = txscript.RawTxInWitnessSignature(tx, sigHash, inputIndex, inputAmount, witnessScript, txscript.SigHashAll, key)
		if err != nil {
			return nil, err
		}
	}

	// Create input script
	var witness wire.TxWitness

	// Push raw signature
	witness = append(witness, sig)

	// Push each chunk of data
	for _, chunk := range chunks {
		witness = append(witness, chunk)
	}

	// Push witness script on the top of the stack
	witness = append(witness, witnessScript)

	// Return serialized P2SH-P2WSH input script
	return witness, nil
}

func buildWitnessScript(pubKey *btcec.PublicKey, chunks [][]byte) ([]byte, error) {
	witnessScript := txscript.NewScriptBuilder()

	// Reverse traversal of chunks such that the stack is popped in the correct order
	for i := len(chunks) - 1; i >= 0; i-- {
		// Hash each chunk of data such that chunks cannot be ordered differently by tx relay nodes or miners
		// This ensures integrity of the data
		witnessScript.AddOp(txscript.OP_HASH160)
		witnessScript.AddData(btcutil.Hash160(chunks[i]))
		witnessScript.AddOp(txscript.OP_EQUALVERIFY)
	}

	// Verify tx signature such that the transaction output cannot be redirected to another address
	// This may not be useful if vout value is equal or close to a dust amount as removing the signature verification would save at most 107 bytes (73 sig + 33 pub + 1 opcode)
	witnessScript.AddData(pubKey.SerializeCompressed())
	witnessScript.AddOp(txscript.OP_CHECKSIG)

	// Return serialized P2SH-P2WSH witness script
	return witnessScript.Script()
}

func buildWitnessProg(witnessScript []byte) []byte {
	// witness program is simply sha256(witnessScript)
	h := sha256.Sum256(witnessScript)
	return h[:]
}

func buildRedeemScript(witnessProg []byte) ([]byte, error) {
	redeemScript := txscript.NewScriptBuilder()

	// redeem script is simply OP_0 0x20 [witness prog]
	redeemScript.AddOp(txscript.OP_0)
	redeemScript.AddData(witnessProg)

	return redeemScript.Script()
}

// WaitPayments waits until all required UTXOs are created on all pre-generated P2SH-P2WSH addresses
func (i *Injection) WaitPayments(onPayment func(addr string, num int)) error {
	var wg sync.WaitGroup
	// Add number of utxos to wait for
	wg.Add(i.NumInputs())

	_, costPerInput, err := i.EstimateCost()
	if err != nil {
		return err
	}

	// Count the number of UTXOs received
	var paymentsReceived int
	var paymentsReceivedMutex sync.Mutex

	for j, address := range i.Addresses {
		go func(addr *btcutil.AddressScriptHash, j int) {
			// Mark job as done
			defer wg.Done()

			for {
				script, err := txscript.PayToAddrScript(addr)
				if err != nil {
					return
				}
				// Electrum servers only accept the hash of the scriptPubKey (in reverse)
				scriptHash := sha256.Sum256(script)
				reversedScriptHash := util.ReverseBytes(scriptHash[:])
				reversedScriptHashHex := hex.EncodeToString(reversedScriptHash)

				// Check all received transactions of a P2SH-P2WSH address
				history, err := electrum.Client.GetHistory(reversedScriptHashHex)
				if err != nil {
					return
				}

				for _, h := range history {
					// Some electrum servers do not support GetTransaction, so we have to decode the raw transaction manually
					rawtx, err := electrum.Client.GetRawTransaction(h.Hash)
					if err != nil {
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
						// Check that the payment address corresponds to the P2SH-P2WSH address and that enough bitcoins were sent
						if bytes.Equal(vout.PkScript, script) && vout.Value >= costPerInput {
							txHash, err := chainhash.NewHashFromStr(h.Hash)
							if err != nil {
								return
							}
							// Add utxo to corresponding P2SH-P2WSH address
							i.Addresses[j].UTXO = wire.NewOutPoint(txHash, uint32(k))

							paymentsReceivedMutex.Lock()
							paymentsReceived++
							paymentsReceivedMutex.Unlock()

							// Event
							onPayment(addr.EncodeAddress(), paymentsReceived)
							return
						}
					}
				}

				// Check for new payments every second
				time.Sleep(time.Second)
			}
		}(address.Address, j)
	}

	// Wait until all payments are received
	wg.Wait()
	return nil
}
