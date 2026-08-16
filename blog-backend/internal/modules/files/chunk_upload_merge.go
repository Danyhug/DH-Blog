package files

import (
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
)

// mergeChunks 按索引顺序把所有分片写入 finalFile，可选同步计算内容哈希。
// 分片在调用前已经过逐索引存在性与大小校验（见 CompleteChunkUpload）。
func mergeChunks(tempDir string, totalChunks int, finalFile *os.File, buffer []byte, hasher hash.Hash) (int64, error) {
	var totalSize int64

	for i := 0; i < totalChunks; i++ {
		chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
		chunk, err := os.Open(chunkFile)
		if err != nil {
			return 0, fmt.Errorf("读取分片 %d 失败: %v", i, err)
		}

		var writer io.Writer = finalFile
		if hasher != nil {
			writer = io.MultiWriter(finalFile, hasher)
		}

		n, err := io.CopyBuffer(writer, chunk, buffer)
		_ = chunk.Close()

		if err != nil {
			return 0, fmt.Errorf("合并分片 %d 失败: %v", i, err)
		}

		totalSize += n
	}

	return totalSize, nil
}
