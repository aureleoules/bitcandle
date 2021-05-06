package retrieval

import (
	"encoding/hex"
	"strings"
)

func P2SHRetrieveData(scriptSigAsm string) ([]byte, error) {
	scriptParts := strings.Split(scriptSigAsm, " ")
	chunks := scriptParts[1 : len(scriptParts)-1]

	var data []byte

	for _, c := range chunks {
		chunkBytes, err := hex.DecodeString(c)
		if err != nil {
			return nil, err
		}

		data = append(data, chunkBytes...)
	}

	return data, nil
}
