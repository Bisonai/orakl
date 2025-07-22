package arr

func SplitByChunkSize[T any](arr []T, chunkSize int) [][]T {
	var chunks [][]T
	for i := 0; i < len(arr); i += chunkSize {
		end := min(i+chunkSize, len(arr))
		chunks = append(chunks, arr[i:end])
	}
	return chunks
}
