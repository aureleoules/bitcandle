package injection

// dataToChunks converts a slice of bytes into multiple chunks of data of the specified size
func dataToChunks(data []byte, size int) [][]byte {
	var chunks [][]byte
	for i := 0; i < len(data); i += size {
		chunks = append(chunks, data[i:min(i+size, len(data))])
	}

	return chunks
}
