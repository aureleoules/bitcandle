package retrieval

import (
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/guumaster/logsymbols"
)

func P2SHRetrieveData(inputs []*wire.TxIn) ([]byte, error) {
	var data []byte

	for _, input := range inputs {
		asmScript, err := txscript.DisasmString(input.SignatureScript)
		if err != nil {
			fmt.Println(logsymbols.Error, "Could not disassemble script signature.")
			os.Exit(1)
		}

		scriptParts := strings.Split(asmScript, " ")
		chunks := scriptParts[1 : len(scriptParts)-1]

		for _, c := range chunks {
			chunkBytes, err := hex.DecodeString(c)
			if err != nil {
				return nil, err
			}

			data = append(data, chunkBytes...)
		}
	}

	return data, nil
}
