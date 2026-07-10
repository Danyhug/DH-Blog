package files

import (
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/sirupsen/logrus"
)

// mergeChunksSequential 顺序合并分片（适用于小文件）
func (h *chunkUploadHandler) mergeChunksSequential(tempDir string, totalChunks int, finalFile *os.File, buffer []byte, hasher hash.Hash) (int64, error) {
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
		chunk.Close()

		if err != nil {
			return 0, fmt.Errorf("合并分片 %d 失败: %v", i, err)
		}

		totalSize += n

		// 定期刷新到磁盘
		if i > 0 && i%50 == 0 {
			finalFile.Sync()
		}
	}

	return totalSize, nil
}

// mergeChunksBuffered 带缓冲的顺序合并（适用于中等文件）
func (h *chunkUploadHandler) mergeChunksBuffered(tempDir string, totalChunks int, finalFile *os.File, buffer []byte, hasher hash.Hash) (int64, error) {
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
		chunk.Close()

		if err != nil {
			return 0, fmt.Errorf("合并分片 %d 失败: %v", i, err)
		}

		totalSize += n

		// 更频繁的磁盘刷新
		if i > 0 && i%25 == 0 {
			finalFile.Sync()
		}
	}

	return totalSize, nil
}

// mergeChunksConcurrent 并发合并分片（适用于大文件）
func (h *chunkUploadHandler) mergeChunksConcurrent(tempDir string, totalChunks int, finalFile *os.File, buffer []byte, hasher hash.Hash) (int64, error) {
	// 动态获取CPU核数并设置工作线程数
	cpuCores := runtime.NumCPU()
	workers := cpuCores * 2 // 通常设置为CPU核数的1-2倍

	// 设置合理的上下限
	if workers < 4 {
		workers = 4 // 最少4个线程
	}
	if workers > 16 {
		workers = 16 // 最多16个线程，避免过度并发
	}

	logrus.Infof("检测到CPU核数: %d, 设置并发工作线程数: %d", cpuCores, workers)

	const batchSize = 100 // 每批处理的分片数

	var totalSize int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// 创建临时文件映射
	tempFiles := make([]string, totalChunks)
	for i := 0; i < totalChunks; i++ {
		tempFiles[i] = filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
	}

	// 分批次处理，避免同时打开过多文件
	for batchStart := 0; batchStart < totalChunks; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > totalChunks {
			batchEnd = totalChunks
		}

		// 为每个批次创建结果通道
		results := make(chan int64, batchEnd-batchStart)
		errors := make(chan error, batchEnd-batchStart)

		// 启动工作线程
		for w := 0; w < workers && batchStart+w < batchEnd; w++ {
			wg.Add(1)
			go func(start, end int) {
				defer wg.Done()

				var batchSize int64
				for i := start; i < end; i += workers {
					chunkFile := tempFiles[i]
					chunk, err := os.Open(chunkFile)
					if err != nil {
						errors <- fmt.Errorf("读取分片 %d 失败: %v", i, err)
						return
					}

					// 获取文件大小
					stat, _ := chunk.Stat()
					chunkSize := stat.Size()

					batchSize += chunkSize
					chunk.Close()
				}

				results <- batchSize
			}(batchStart+w, batchEnd)
		}

		// 等待批次完成
		wg.Wait()
		close(results)
		close(errors)

		// 处理结果
		for size := range results {
			mu.Lock()
			totalSize += size
			mu.Unlock()
		}

		// 处理错误
		select {
		case err := <-errors:
			if firstErr == nil {
				firstErr = err
			}
		default:
		}

		if firstErr != nil {
			return 0, firstErr
		}
	}

	// 重新顺序写入（并发阶段只计算大小，这里真正写入）
	// 注：由于并发写入到同一个文件的复杂性，这里使用优化的顺序写入
	var writer io.Writer = finalFile
	if hasher != nil {
		writer = io.MultiWriter(finalFile, hasher)
	}

	for i := 0; i < totalChunks; i++ {
		chunkFile := tempFiles[i]
		chunk, err := os.Open(chunkFile)
		if err != nil {
			return 0, fmt.Errorf("读取分片 %d 失败: %v", i, err)
		}

		_, err = io.CopyBuffer(writer, chunk, buffer)
		chunk.Close()

		if err != nil {
			return 0, fmt.Errorf("合并分片 %d 失败: %v", i, err)
		}

		if i > 0 && i%100 == 0 {
			finalFile.Sync()
		}
	}

	return totalSize, nil
}
