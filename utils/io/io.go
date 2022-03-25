package io_utils

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"go-common/utils/core"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// RemoveCommentReader remove comment from the io.Reader
// start with // to end of line
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
func PathExists(path string) bool {
	_, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return err == nil
}

// IsFile 是否是文件
func IsFile(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	return stat.Mode().IsRegular()
}

// IsDir 是否是目录
func IsDir(dir string) bool {
	stat, err := os.Stat(dir)
	if err != nil {
		return false
	}

	return stat.Mode().IsDir()
}

// IsExecutable 是否是可执行程序，windows下，只要文件存在，就返回true
func IsExecutable(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil || fileInfo.Mode().IsDir() {
		return false
	}

	if runtime.GOOS == "windows" {
		return true
	}

	if fileInfo.Mode()&0111 != 0 {
		return true
	}

	return false
}

func FileSize(path string) int64 {
	stat, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return stat.Size()
}

func FileMode(path string) os.FileMode {
	if stat, err := os.Stat(path); err != nil {
		return 0
	} else {
		return stat.Mode()
	}
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

func MustMkdirAll(path string, perm os.FileMode) error {
	// 目录存在
	if PathExists(path) {
		if IsDir(path) {
			if perm > 0 {
				return os.Chmod(path, perm)
			} else {
				return nil
			}
		} else {
			return fmt.Errorf("can not make directory, because the path of \"%s\" is a file", path)
		}
	}

	return os.MkdirAll(path, perm)
}

// Md5 文件的MD5值
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

func EnvPaths() []string {
	path := os.Getenv("PATH")
	return filepath.SplitList(path)
}

// Which 类似windows/linux中的witch、where、whereis指令
// 如果只需要返回一个的话 使用系统的exec.LookPath()
func Which(filename string) []string {
	var list []string
	for _, p := range EnvPaths() {
		if !IsDir(p) {
			continue
		}
		fileList, err := ioutil.ReadDir(p)
		if err != nil {
			continue
		}

		for _, f := range fileList {
			path := filepath.Join(p, f.Name())
			if runtime.GOOS == "windows" {
				if strings.EqualFold(f.Name(), filename) {
					list = append(list, path)
				}
			} else if f.Name() == filename {
				list = append(list, path)
			}
		}
	}
	return list
}

// Unmount (强制)umount一个目录
func Unmount(path string, force bool) error {
	cmd := exec.Command("umount", core.If(force, "-f", ""), path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			output = bytes.TrimRight(output, "\n")
			msg := err.Error() + ": " + string(output)
			err = errors.New(msg)
		}
		return err
	}
	return nil
}

// MakePathFromRelative 当path是相对路径是, 添加prefix在path之前, 如果path是绝对路径, 直接返回path
// prefix默认为程序当前目录
func MakePathFromRelative(prefix, path string) string {
	if filepath.IsAbs(path) {
		return path
	} else {
		if prefix == "" {
			prefix = GetCurrentDir()
		}
		return filepath.Join(prefix, path)
	}
}
