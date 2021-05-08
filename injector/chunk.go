package injector

import (
	"bytes"

	"github.com/aureleoules/bitcandle/consensus"
	"github.com/aureleoules/bitcandle/util"
)

// dataToParts converts a slice of bytes into multiple chunks of data of the specified size
func dataToChunks(data []byte, size int) [][]byte {
	var chunks [][]byte

	for i := 0; i < len(data); i += size {
		chunks = append(chunks, data[i:util.Min(i+size, len(data))])
	}

	return chunks
}

// dataToChunks converts a slice of bytes into multiple chunks of data of the specified size
func dataToParts(data []byte) [][]byte {
	var chunks [][]byte
	var index byte = 0x00

	for i := 0; i < len(data); i += consensus.P2SHInputDataLimit - 1 {
		var buffer bytes.Buffer
		buffer.WriteByte(index)
		buffer.Write(data[i:util.Min(i+consensus.P2SHInputDataLimit-1, len(data))])

		chunks = append(chunks, buffer.Bytes())
		index++
	}

	return chunks
}
