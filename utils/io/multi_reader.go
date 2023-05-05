package ioUtils

import (
	"github.com/pkg/errors"
	"go.uber.org/multierr"
	"io"
)

type MultiReadSeeker struct {
	readers []io.ReadSeekCloser

	starts []int64
	sizes  []int64

	position int64
}

var _ io.ReadSeekCloser = (*MultiReadSeeker)(nil)

// NewMultiReadSeeker 多个reader组成一个reader
//  1. reader的长度必须是固定的，否则会出现问题
//  2. 从传入的reader的当前位置（Seek(0, io.SeekCurrent)）开始读取，如果需要从头开始读取，需要在传入前就对reader.Seek到0
//
// NewMultiReadSeeker combine multiple readers to one reader
//  1. the length of readers must be fixed, otherwise it will cause problems
//  2. read from the current position of the incoming reader (Seek(0, io.SeekCurrent)), if you need to read from the beginning, you need to Seek to 0 before passing in
func NewMultiReadSeeker(readers ...io.ReadSeekCloser) *MultiReadSeeker {
	return &MultiReadSeeker{readers: readers}
}

func (m *MultiReadSeeker) prepare() error {
	if len(m.sizes) == len(m.readers) {
		return nil
	}

	// 初始化
	m.starts = make([]int64, 0, len(m.readers))
	m.sizes = make([]int64, 0, len(m.readers))

	for i := 0; i < len(m.readers); i++ {
		// 获取当前位置
		current, err := m.readers[i].Seek(0, io.SeekCurrent)
		if err != nil {
			return errors.WithStack(err)
		}
		m.starts = append(m.starts, current)

		// 获取文件末尾位置
		end, err := m.readers[i].Seek(0, io.SeekEnd)
		if err != nil {
			return errors.WithStack(err)
		}

		_, err = m.readers[i].Seek(current, io.SeekStart)
		if err != nil {
			return errors.WithStack(err)
		}

		m.sizes = append(m.sizes, end-current)
	}

	return nil
}

func (m *MultiReadSeeker) Size() int64 {
	_ = m.prepare()

	var total int64
	for i := 0; i < len(m.sizes); i++ {
		total += m.sizes[i]
	}

	return total
}

// Seek 实现io.Seeker接口
// Seek implements io.Seeker interface
func (m *MultiReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if err := m.prepare(); err != nil {
		return 0, err
	}

	total := m.Size()
	var abs int64
	switch whence {
	case io.SeekStart:
		abs = offset
	case io.SeekCurrent:
		abs = m.position + offset
	case io.SeekEnd:
		abs = total + offset
	default:
		return 0, errors.New("MultipartFileReader.Reader.Seek: invalid whence")
	}

	if abs < 0 || abs > total {
		return 0, errors.New("MultipartFileReader.Reader.Seek: negative position, or position out of range")
	}

	m.position = abs
	return abs, nil
}

// Read 实现io.Reader接口
// Read implements io.Reader interface
func (m *MultiReadSeeker) Read(p []byte) (n int, err error) {
	if err = m.prepare(); err != nil {
		return 0, err
	}

	current := m.position
	for i := 0; i < len(m.readers); i++ {
		// 查找当前位置在哪个reader中
		// Find the reader where the current position is located
		if current < m.sizes[i] {
			// 将该reader移动到current位置
			// Move the reader to the current position
			_, err = m.readers[i].Seek(m.starts[i]+current, io.SeekStart)
			if err != nil {
				return 0, errors.WithStack(err)
			}

			// 只读取当前reader的数据就返回，不跨reader读取
			//（因为Read方法的n会返回真实读取的数据长度，程序进行下一次读取时，Seek会自动跳转到下一个reader）
			// 这样做的好处是减少逻辑复杂度，不用考虑在读取时跨reader的情况；但是缺点是，如果该reader余量不足填充p时，会需要多次调用Read方法
			// Only return the data of the current reader, do not read across the reader
			// (Because the n of the Read method will return the length of the actually read data, the program will automatically jump to the next reader when reading next time)
			// The advantage of doing this is to reduce the complexity of the logic, and you don’t need to consider the situation of reading across the reader when reading;
			// But the disadvantage is that if the remaining amount of the reader is insufficient to fill p, the Read method will need to be called multiple times
			n, err = m.readers[i].Read(p)
			if err == io.EOF {
				err = nil
			}

			m.position += int64(n)
			return
		}

		current -= m.sizes[i]
	}

	return 0, io.EOF
}

func (m *MultiReadSeeker) Close() error {
	var err error
	for _, reader := range m.readers {
		err = multierr.Append(err, reader.Close())
	}
	return err
}

type MultiReadCloser struct {
	readers []io.ReadCloser
}

var _ io.ReadCloser = (*MultiReadCloser)(nil)

// NewMultiReadCloser 多个reader组成一个reader，只提供读取功能。
//
//	读取时，会依次读取每个reader，直到读取到EOF，然后关闭该reader，再读取下一个reader。
func NewMultiReadCloser(readers ...io.ReadCloser) *MultiReadCloser {
	return &MultiReadCloser{readers: readers}
}

func (m *MultiReadCloser) Read(p []byte) (n int, err error) {
	if len(m.readers) == 0 {
		return 0, io.EOF
	}

	n, err = m.readers[0].Read(p)
	if err == io.EOF {
		err = m.readers[0].Close()
		m.readers = m.readers[1:]

		if err != nil {
			return
		}
	}

	return
}

func (m *MultiReadCloser) Close() error {
	var err error
	for _, reader := range m.readers {
		err = multierr.Append(err, reader.Close())
	}

	return err
}
