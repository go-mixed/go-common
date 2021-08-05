package utils

import (
	"fmt"
	"go-common/utils/text"
	"io/ioutil"
)

func LoadSettings(v interface{}, filename string) error {
	content, err := ioutil.ReadFile(filename)

	if err != nil {
		return fmt.Errorf("read settings file error: %w", err)
	}

	err = text_utils.JsonUnmarshalFromBytes(content, v)

	if err != nil {
		return fmt.Errorf("read settings json error: %w", err)
	}

	return nil
}

func WriteSettings(v interface{}, filename string) error {
	j, err := text_utils.JsonMarshalToBytes(v)
	if err != nil {
		return fmt.Errorf("marshal settings json error: %w", err)
	}

	err = ioutil.WriteFile(filename, j, 0o664)
	if err != nil {
		return fmt.Errorf("write settings file error: %w", err)
	}

	return nil
}

