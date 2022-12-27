package text_utils

import (
	"bytes"
	"encoding/gob"
	"github.com/pkg/errors"
)

func GobEncode(obj any) ([]byte, error) {
	buf := &bytes.Buffer{}
	g := gob.NewEncoder(buf)
	if err := g.Encode(obj); err != nil {
		return nil, errors.WithStack(err)
	}

	return buf.Bytes(), nil
}

func GobDecode(buf []byte, v any) error {
	reader := bytes.NewReader(buf)
	g := gob.NewDecoder(reader)
	return errors.WithStack(g.Decode(v))
}

func GobListDecode(buf [][]byte, v any) error {
	return ListDecode(GobDecode, buf, v)
}
