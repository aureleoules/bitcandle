package injection

import (
	"github.com/aureleoules/bitcandle/consensus"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

func P2SHScriptAddr(data []byte, pubKey *btcec.PublicKey, network *chaincfg.Params) (*btcutil.AddressScriptHash, error) {
	// Split data into chunks of 520 bytes
	// This is the maximum of data that can be pushed on the stack at a time
	chunks := dataToChunks(data, consensus.P2SHPushDataLimit)

	// Build redeem script
	redeemScript, err := buildRedeemScript(pubKey, chunks)
	if err != nil {
		return nil, err
	}

	// Hash redeemscript to build the scriptHash address
	return btcutil.NewAddressScriptHash(redeemScript, network)
}

func P2SHBuildTX(data []byte, prevOut *wire.OutPoint, txOut *wire.TxOut, key *btcec.PrivateKey, network *chaincfg.Params) (*wire.MsgTx, error) {
	tx := wire.NewMsgTx(wire.TxVersion)
	tx.AddTxOut(txOut)

	txIn := wire.NewTxIn(prevOut, nil, nil)

	tx.AddTxIn(txIn)

	chunks := dataToChunks(data, consensus.P2SHPushDataLimit)
	tx.TxIn[0].SignatureScript, _ = buildSignatureScript(tx, chunks, key)

	return tx, nil
}

func buildSignatureScript(tx *wire.MsgTx, chunks [][]byte, key *btcec.PrivateKey) ([]byte, error) {
	redeemScript, err := buildRedeemScript(key.PubKey(), chunks)
	if err != nil {
		return nil, err
	}

	// Sign transaction pre-image
	sig, err := txscript.RawTxInSignature(tx, 0, redeemScript, txscript.SigHashAll, key)
	if err != nil {
		return nil, err
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
