package utils

import (
	"crypto/md5"
	"fmt"
	"hash"
	"io"
	"os"
	"strings"
)

type MultipartFileReader struct {
	paths  []string
	sizes   []int64
	totalSize int64
	start  int64
	length int64

	readBytes int64

	files []*os.File
	hash     hash.Hash
	checksums []byte
}

// NewMultipartFileReader 多个文件分块组成一个文件的reader
// eg: 读取整个文件 NewMultipartFileReader(["file1", "file2"], 0, -1, -1)
// eg: 偏移值从30开始, 读取100长度 读取文件 NewMultipartFileReader(["file1", "file2"], 30, 100, -1)
// parameter: expectedTotalSize: 期望得到文件大小总数, 可以检查分块文件大小是否完整, 如果不检查 则设置为-1
func NewMultipartFileReader(paths []string, start int64, length int64, expectedTotalSize int64) (*MultipartFileReader, error) {
	sizes := make([]int64, len(paths))

	var total int64
	for i, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("path \"%s\" error: %w", path, err)
		}

		sizes[i] = stat.Size()
		total += sizes[i]
	}

	if expectedTotalSize != -1 && expectedTotalSize != total {
		return nil, fmt.Errorf("expectedTotalSize: %d is not equal to real total size: %d", expectedTotalSize, total)
	}

	if start < 0 {
		start = 0
	}
	// length 为 -1 或超出文件大小 则设置到结尾
	if length < 0 || start + length > total  {
		length = total - start
	}

	return &MultipartFileReader{
		paths:  paths,
		sizes:  sizes,
		start:  start,
		length: length,
		totalSize: total,
		files: make([]*os.File, len(paths)),
		hash:     md5.New(),
	}, nil
}

func (r *MultipartFileReader) openFile(index int) (*os.File,  error) {
	if r.files[index] == nil {
		file, err := os.Open(r.paths[index])
		if err != nil {
			return nil, err
		}
		r.files[index] = file
	}

	return r.files[index], nil
}

func (r *MultipartFileReader) closeFile(index int) error {
	if r.files[index] != nil {
		err := r.files[index].Close()
		r.files[index] = nil
		return err
	}
	return nil
}

func (r *MultipartFileReader) Read(buf []byte) (int, error) {

	var currentRead = 0
	var n = 0
	var file *os.File
	var err error
	var fileIndex = 0
	var fileStart int64 = 0

	for {
		// 还能够读取的size
		canReadSize := len(buf) - currentRead
		remain := r.length - r.readBytes
		// 如果没有剩余的待读的 返回 EOF 并计算整体的md5
		if remain <= 0 {
			r.checksums = r.hash.Sum(nil)
			return currentRead, io.EOF
		} else if int(remain) < canReadSize { // 剩余的比buf能读的都要小
			canReadSize = int(remain)
		}
		// 没有可读的
		if canReadSize == 0 {
			return currentRead, nil
		}

		// 查找该读取哪个文件, 并设置好文件seek
		for ;fileIndex < len(r.sizes); {
			var startOffset = r.readBytes + r.start
			// offset在文件范围内
			if startOffset >= fileStart && startOffset < fileStart+r.sizes[fileIndex] {
				file, err = r.openFile(fileIndex)
				if err != nil {
					return currentRead, err
				}
				offset := startOffset - fileStart
				_, err = file.Seek(offset, io.SeekStart) // 移动文件偏移值
				if err != nil {
					return currentRead, err
				}
				break
			}
			fileStart += r.sizes[fileIndex]
			fileIndex++
		}

		// 超出文件范围了
		if fileIndex >= len(r.files) {
			return 0, io.ErrUnexpectedEOF
		}

		n, err = file.Read(buf[currentRead : currentRead+canReadSize])
		if err == io.EOF { // 如果文件结束 则读取下一个文件
			r.closeFile(fileIndex)
		} else if err != nil {
			return currentRead, err
		}

		// 更新md5
		r.hash.Write(buf[currentRead:currentRead+n])
		currentRead += n
		r.readBytes += int64(n)

		// 已经读完
		if currentRead >= len(buf) {
			break
		}
	}

	return currentRead, nil
}

// DryRead 读取所有数据, 但是不返回文件内容, 文件大会很耗时
// 通过此方式可以在执行 DryRead 后调用 Checksums 得到文件的md5
func (r *MultipartFileReader) DryRead() (int64, error) {
	buf := make([]byte, 1024)
	var size int64 = 0
	for {
		n, err := r.Read(buf)
		size += int64(n)

		if err == io.EOF {
			break
		} else if err != nil {
			return size, err
		}
	}

	return size, nil
}


// Checksums 需要read结束之后才能得到正确的md5
func (r *MultipartFileReader) Checksums(into []byte) []byte {
	if r.checksums != nil {
		return r.checksums
	}
	return r.hash.Sum(into)
}

func (r *MultipartFileReader) Close() error {
	var errs []string
	for i := range r.files {
		err := r.closeFile(i)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return fmt.Errorf(strings.Join(errs, "\n"))
}

