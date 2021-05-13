package utils

import (
	"bytes"
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

func GetCurrentDir() string {
	// 读取当前执行文件的目录
	_path, err := os.Executable()
	if err != nil {
		fmt.Print("read current path error")
		return ""
	}
	return filepath.Dir(_path)
}
