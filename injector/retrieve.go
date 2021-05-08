package injector

import (
	"bytes"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// P2SHRetrieveData takes a list of transaction inputs and decodes the signature scripts that contain the file
func P2SHRetrieveData(rawTxBytes []byte) ([]byte, error) {
	var tx wire.MsgTx

	err := tx.Deserialize(bytes.NewReader(rawTxBytes))
	if err != nil {
		return nil, errors.New("could not decode transaction")
	}

	var data []byte

	for _, input := range tx.TxIn {
		// Disassemble the signature script to make parsing easier
		asmScript, err := txscript.DisasmString(input.SignatureScript)
		if err != nil {
			return nil, errors.New("could not disassemble script signature")
		}

		scriptParts := strings.Split(asmScript, " ")
		if len(scriptParts) <= 2 {
			return nil, errors.New("invalid signature script")
		}
		chunks := scriptParts[1 : len(scriptParts)-1]

		for _, c := range chunks {
			chunkBytes, err := hex.DecodeString(c)
			if err != nil {
				return nil, err
			}

			// Concat each chunk of 520 bytes (max)
			data = append(data, chunkBytes...)
		}
	}

	return data, nil
}
