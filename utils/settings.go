package utils

import (
	"fmt"
	"go-common/utils/text"
	"io/ioutil"
)

func LoadSettings(v interface{}, filename string) error {
	content, err := ioutil.ReadFile(filename)

	if err != nil {
		return fmt.Errorf("settings file error: %w", err)
	}

	err = text_utils.JsonUnmarshalFromBytes(content, v)

	if err != nil {
		return fmt.Errorf("settings json error: %w", err)
	}

	return nil
}
