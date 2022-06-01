package io_utils

import (
	"crypto/md5"
	"errors"
	"fmt"
	"go.uber.org/multierr"
	"hash"
	"io"
	"io/ioutil"
	"os"
)

type MultipartFileReader struct {
	paths         []string
	sizes         []int64
	totalSize     int64
	position      int64
	readingLength int64

	files []*os.File

	hash      hash.Hash
	checksums []byte
}

// NewMultipartFileReader 多个文件分块组成一个文件的reader
// 注意, 如果在读取中修改了某分片文件的长度, 最终读取得到的数据可能不符合预期
// eg: 偏移值从30开始, 读取100长度
// reader := NewMultipartFileReader(["file1", "file2"])
// reader.Seek(30)
// var buf = make([]byte, 100)
// reader.reader(buf)
func NewMultipartFileReader(paths []string) (*MultipartFileReader, error) {
	sizes := make([]int64, len(paths))

	var totalSize int64
	for i, path := range paths {
		stat, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("path \"%s\" error: %w", path, err)
		} else if !IsFile(path) {
			return nil, fmt.Errorf("path \"%s\" is not a file", path)
		}

		sizes[i] = stat.Size()
		totalSize += sizes[i]
	}

	return &MultipartFileReader{
		paths:     paths,
		sizes:     sizes,
		position:  0,
		totalSize: totalSize,
		files:     make([]*os.File, len(paths)),
		hash:      md5.New(),
	}, nil
}

func (r *MultipartFileReader) openFile(index int) (*os.File, error) {
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

func (r *MultipartFileReader) Size() int64 {
	return r.totalSize
}

func (r *MultipartFileReader) ReadingLength() int64 {
	return r.readingLength
}

func (r *MultipartFileReader) Seek(offset int64, whence int) (int64, error) {

	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = r.position + offset
	case io.SeekEnd:
		//abs = r.totalSize + offset
		return 0, errors.New("MultipartFileReader.Reader.Seek: unsupported io.SeekEnd")
	default:
		return 0, errors.New("MultipartFileReader.Reader.Seek: invalid whence")
	}

	if abs < 0 {
		return 0, errors.New("MultipartFileReader.Reader.Seek: negative position")
	}

	r.position = abs
	return abs, nil
}

func (r *MultipartFileReader) Position() int64 {
	return r.position
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
		remain := r.totalSize - r.position
		// 如果没有剩余的待读的 返回 EOF 并计算整体的md5
		if remain <= 0 {
			//r.checksums = r.hash.Sum(nil)
			return currentRead, io.EOF
		} else if int(remain) < canReadSize { // 剩余的比buf能读的都要小
			canReadSize = int(remain)
		}
		// 没有可读的
		if canReadSize == 0 {
			return currentRead, nil
		}

		// 查找该读取哪个文件, 并设置好文件seek
		for fileIndex < len(r.sizes) {
			// offset在文件范围内
			if r.position >= fileStart && r.position < fileStart+r.sizes[fileIndex] {
				file, err = r.openFile(fileIndex)
				if err != nil {
					return currentRead, err
				}
				offset := r.position - fileStart
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
		r.position += int64(n) // 更新position位置

		if err == io.EOF { // 如果文件结束 则读取下一个文件
			r.closeFile(fileIndex)
		} else if err != nil {
			return currentRead, err
		}

		// 更新md5
		r.hash.Write(buf[currentRead : currentRead+n])
		currentRead += n
		r.readingLength += int64(n)

		// 已经读完
		if currentRead >= len(buf) {
			break
		}
	}

	return currentRead, nil
}

// DryRead readSize 设置为-1表示从当前position开始, 直到读取到结尾; >= 0 则从当前position开始, 读取指定数量的数据
// 不返回文件内容, 如果文件比较大会很耗时
// 通过此方式可以在执行 DryRead 后调用 Checksums 得到文件的md5
func (r *MultipartFileReader) DryRead(readSize int64) (int64, error) {
	if readSize == -1 {
		return io.Copy(ioutil.Discard, r)
	} else {
		return io.CopyN(ioutil.Discard, r, readSize)
	}
}

// Checksums 需要read结束之后才能得到正确的md5
func (r *MultipartFileReader) Checksums(into []byte) []byte {
	if r.checksums != nil {
		return r.checksums
	}
	return r.hash.Sum(into)
}

// Close 关闭文件
func (r *MultipartFileReader) Close() error {
	var errs error
	for i := range r.files {
		err := r.closeFile(i)
		if err != nil {
			errs = multierr.Append(errs, err)
		}
	}
	return errs
}
