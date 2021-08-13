package io_utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// RemoveCommentReader remove comment from the io.Reader
func RemoveCommentReader(reader io.Reader) (newReader io.Reader) {

	bs, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	s := string(bs)
	re1 := regexp.MustCompile(`(?im)^\s+\/\/.*$`) // 整行注释
	s = re1.ReplaceAllString(s, "")
	re2 := regexp.MustCompile(`(?im)\/\/[^"\[\]]+$`) // 行末
	s = re2.ReplaceAllString(s, "")
	newReader = strings.NewReader(s)
	return
}

func BytesToReaderWithCloser(_bytes []byte) io.ReadCloser {
	return ioutil.NopCloser(bytes.NewBuffer(_bytes))
}

func ReadAndRestoreReader(reader *io.ReadCloser) []byte {
	if reader == nil {
		return nil
	}

	_bytes, _ := ioutil.ReadAll(*reader)
	// close original reader
	(*reader).Close()

	// Restore the io.ReadCloser to its original state
	*reader = BytesToReaderWithCloser(_bytes)

	return _bytes
}

// GetCurrentDir 得到当前执行文件的路径
// 为了方便调试, 也可以设置环境变量 CURRENT_DIRECTORY 来替代真正的文件路径
func GetCurrentDir() string {
	// 为了方便调试
	p := os.Getenv("CURRENT_DIRECTORY")
	if p != "" {
		return p
	}
	// 读取当前执行文件的目录
	_path, err := os.Executable()
	if err != nil {
		fmt.Print("read current path error")
		return ""
	}
	return filepath.Dir(_path)
}

// PathExists 检测路径是否存在, 不区分文件/文件夹
func PathExists(name string) bool {
	_, err := os.Stat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return err == nil
}

// IsFile 是否是文件
func IsFile(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}

	return stat.Mode().IsRegular()
}

// IsDir 是否是目录
func IsDir(name string) bool {
	stat, err := os.Stat(name)
	if err != nil {
		return false
	}

	return stat.Mode().IsDir()
}

func FileSize(name string) int64 {
	stat, err := os.Stat(name)
	if err != nil {
		return 0
	}
	return stat.Size()
}

// MoveFile will work moving file between folders.
// GoLang: os.Rename() give error "invalid cross-device link" for Docker container with Volumes.
func MoveFile(sourcePath, destPath string) error {
	if err := CopyFile(sourcePath, destPath); err != nil {
		return err
	}

	// The copy was successful, so now delete the original file
	if err := os.Remove(sourcePath); err != nil {
		return fmt.Errorf("failed removing original file: %s", err)
	}

	return nil
}

// CopyFile copy sourcePath to destPath with the same perm as the source file
func CopyFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %s", err)
	}
	defer inputFile.Close()

	fi, err := inputFile.Stat()
	if err != nil {
		return err
	}

	//  perm as same as source file
	flag := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
	perm := fi.Mode() & os.ModePerm
	outputFile, err := os.OpenFile(destPath, flag, perm)
	if err != nil {
		return fmt.Errorf("couldn't open dest file: %s", err)
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return fmt.Errorf("writing to output file failed: %s", err)
	}

	if err := outputFile.Sync(); err != nil {
		return fmt.Errorf("failed sync dest file: %s", err)
	}

	return nil
}

func Md5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}
	hashInBytes := hash.Sum(nil)[:16]
	return hex.EncodeToString(hashInBytes), nil
}