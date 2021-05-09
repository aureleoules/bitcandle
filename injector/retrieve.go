package injector

import (
	"bytes"
	"errors"

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
		// Skip signature and witness script
		for i := 1; i < len(input.Witness)-1; i++ {
			data = append(data, input.Witness[i]...)
		}
	}

	return data, nil
}
