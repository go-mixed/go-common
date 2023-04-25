package logger

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
	"time"
)

type encoderFunc func() zapcore.Encoder

func (f encoderFunc) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	return f().AddArray(key, marshaler)
}

func (f encoderFunc) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	return f().AddObject(key, marshaler)
}

func (f encoderFunc) AddBinary(key string, value []byte) {
	f().AddBinary(key, value)
}

func (f encoderFunc) AddByteString(key string, value []byte) {
	f().AddByteString(key, value)
}

func (f encoderFunc) AddBool(key string, value bool) {
	f().AddBool(key, value)
}

func (f encoderFunc) AddComplex128(key string, value complex128) {
	f().AddComplex128(key, value)
}

func (f encoderFunc) AddComplex64(key string, value complex64) {
	f().AddComplex64(key, value)
}

func (f encoderFunc) AddDuration(key string, value time.Duration) {
	f().AddDuration(key, value)
}

func (f encoderFunc) AddFloat64(key string, value float64) {
	f().AddFloat64(key, value)
}

func (f encoderFunc) AddFloat32(key string, value float32) {
	f().AddFloat32(key, value)
}

func (f encoderFunc) AddInt(key string, value int) {
	f().AddInt(key, value)
}

func (f encoderFunc) AddInt64(key string, value int64) {
	f().AddInt64(key, value)
}

func (f encoderFunc) AddInt32(key string, value int32) {
	f().AddInt32(key, value)
}

func (f encoderFunc) AddInt16(key string, value int16) {
	f().AddInt16(key, value)
}

func (f encoderFunc) AddInt8(key string, value int8) {
	f().AddInt8(key, value)
}

func (f encoderFunc) AddString(key, value string) {
	f().AddString(key, value)
}

func (f encoderFunc) AddTime(key string, value time.Time) {
	f().AddTime(key, value)
}

func (f encoderFunc) AddUint(key string, value uint) {
	f().AddUint(key, value)
}

func (f encoderFunc) AddUint64(key string, value uint64) {
	f().AddUint64(key, value)
}

func (f encoderFunc) AddUint32(key string, value uint32) {
	f().AddUint32(key, value)
}

func (f encoderFunc) AddUint16(key string, value uint16) {
	f().AddUint16(key, value)
}

func (f encoderFunc) AddUint8(key string, value uint8) {
	f().AddUint8(key, value)
}

func (f encoderFunc) AddUintptr(key string, value uintptr) {
	f().AddUintptr(key, value)
}

func (f encoderFunc) AddReflected(key string, value interface{}) error {
	return f().AddReflected(key, value)
}

func (f encoderFunc) OpenNamespace(key string) {
	f().OpenNamespace(key)
}

func (f encoderFunc) Clone() zapcore.Encoder {
	return f().Clone()
}

func (f encoderFunc) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	return f().EncodeEntry(entry, fields)
}
