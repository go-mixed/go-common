package ioUtils

import (
	"bytes"
	"io"
)

// BytesToReadCloser []byte转换为一个带close的reader
func BytesToReadCloser(_bytes []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewBuffer(_bytes))
}

// ReadAndRestoreReader 读取reader的全部内容，并重新赋值给一个[]byte的reader，只建议用于小文件
//
//	应用场景：
//	response, _ = http.Get(...)
//	content = ReadAndRestoreReader(&response.Body) // 比如在代理环境：即获得了内容，又让下文可以继续操作response.Body
//
//	response.Body.Read(...)
//	response.Body.Close()
func ReadAndRestoreReader(reader *io.ReadCloser) []byte {
	if reader == nil {
		return nil
	}

	_bytes, _ := io.ReadAll(*reader)
	// close original reader
	(*reader).Close()

	// Restore the io.ReadCloser to its original state
	*reader = BytesToReadCloser(_bytes)

	return _bytes
}
