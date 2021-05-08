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
