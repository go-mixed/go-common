package utils

import (
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"io/ioutil"
)

func LoadSettings(v interface{}, filename string) error {
	content, err := ioutil.ReadFile(filename)

	if err != nil {
		return fmt.Errorf("settings file error: %w", err)
	}

	var json = jsoniter.ConfigCompatibleWithStandardLibrary

	err = json.Unmarshal(content, v)

	if err != nil {
		return fmt.Errorf("settings json error: %w", err)
	}

	return nil
}
