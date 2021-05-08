package injector

import (
	"encoding/hex"
	"errors"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// P2SHRetrieveData takes a list of transaction inputs and decodes the signature scripts that contain the file
func P2SHRetrieveData(inputs []*wire.TxIn) ([]byte, error) {
	var data []byte

	for _, input := range inputs {
		// Disassemble the signature script to make parsing easier
		asmScript, err := txscript.DisasmString(input.SignatureScript)
		if err != nil {
			return nil, errors.New("could not disassemble script signature")
		}

		scriptParts := strings.Split(asmScript, " ")
		chunks := scriptParts[1 : len(scriptParts)-1]

		skipped := false

		for _, c := range chunks {
			chunkBytes, err := hex.DecodeString(c)
			if err != nil {
				return nil, err
			}

			if !skipped {
				// Make sure to skip the first byte of each input's signature script
				// as the input index is store there to build a different redeem script for each utxo
				data = append(data, chunkBytes[1:]...)
				skipped = true
			} else {
				// Concat each chunk of 520 bytes (max)
				data = append(data, chunkBytes...)
			}

		}
	}

	return data, nil
}
