package text_utils

import (
	"github.com/pkg/errors"
	"strconv"
)

type JsonBool bool

func (b *JsonBool) UnmarshalJSON(data []byte) error {
	asString := string(data)
	_b, err := strconv.ParseBool(asString)
	*b = JsonBool(_b)
	if err != nil {
		return errors.Errorf("boolean unmarshal of json error: %w", err)
	}
	return err
}

func (b JsonBool) AsBool() bool {
	return bool(b)
}
