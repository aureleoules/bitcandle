package injection

import (
	"bytes"
	"math"

	"github.com/aureleoules/bitcandle/consensus"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

type P2SHInjectionAddress struct {
	Address *btcutil.AddressScriptHash
	UTXO    *wire.OutPoint
	Chunks  [][]byte
}

type P2SHInjection struct {
	Network   *chaincfg.Params
	FeeRate   int
	Addresses []*P2SHInjectionAddress

	parts      [][]byte
	privateKey *btcec.PrivateKey
}

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

		injection.Addresses = append(injection.Addresses, &P2SHInjectionAddress{
			Address: addr,
			Chunks:  chunks,
		})
	}

	return &injection, nil
}
func (i *P2SHInjection) NumInputs() int {
	return len(i.parts)
}

func (i *P2SHInjection) EstimateCost() (float64, float64, error) {
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
	dummyTx.Serialize(&dummyTxBytes)

	// Count tx bytes and estimate cost of transaction
	costSats := float64(len(dummyTxBytes.Bytes())*i.FeeRate + consensus.P2PKHDustLimit)

	costPerInput := math.Ceil(costSats/float64(i.NumInputs())) / consensus.BTCSats

	return float64(costSats) / consensus.BTCSats, costPerInput, nil
}

func (i *P2SHInjection) BuildTX(txOut *wire.TxOut) (*wire.MsgTx, error) {
	return i.buildTX(txOut, false)
}

func (i *P2SHInjection) buildTX(txOut *wire.TxOut, dummy bool) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(txOut)

	for k, addr := range i.Addresses {
		if dummy {
			addr.UTXO = wire.NewOutPoint(chaincfg.MainNetParams.GenesisHash, 0)
		}

		txIn := wire.NewTxIn(addr.UTXO, nil, nil)
		tx.AddTxIn(txIn)

		var err error
		tx.TxIn[k].SignatureScript, err = buildSignatureScript(tx, addr.Chunks, i.privateKey, dummy)
		if err != nil {
			return nil, err
		}
	}

	return tx, nil
}

func buildSignatureScript(tx *wire.MsgTx, chunks [][]byte, key *btcec.PrivateKey, dummy bool) ([]byte, error) {
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
		sig, err = txscript.RawTxInSignature(tx, 0, redeemScript, txscript.SigHashAll, key)
		if err != nil {
			return nil, err
		}
	}

	inputScript := txscript.NewScriptBuilder()

	inputScript.AddData(sig)

	for _, chunk := range chunks {
		inputScript.AddData(chunk)
	}

	inputScript.AddData(redeemScript)

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

	return redeemScript.Script()
}
