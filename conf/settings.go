package conf

import (
	"bytes"
	"encoding/json"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

// LoadSettings 读取JSON/YAML格式的配置, v必须为指针
// 可以传入多个文件，json、yaml可以混用，后面文件的配置会覆盖前面的配置
// json 能被yaml.Unmarshal解析，但是c风格注释会被解析成kv值
// 支持(https://github.com/go-playground/validator)的校验格式，比如：struct {Url string `yaml:"url" validate:"required,url,min=5,max=256"`}
func LoadSettings(v any, filenames ...string) error {
	for _, filename := range filenames {
		ext := filepath.Ext(filename)
		if content, err := os.ReadFile(filename); err != nil {
			return errors.Errorf("read settings file error: %w", err)
		} else if strings.EqualFold(ext, ".json") ||
			strings.EqualFold(ext, ".json5") ||
			strings.EqualFold(ext, ".yaml") ||
			strings.EqualFold(ext, ".yml") {
			if err = yaml.Unmarshal(content, v); err != nil {
				return errors.Errorf("unmarshal settings file \"%s\" error: %w", filename, err)
			}
		} else {
			return errors.Errorf("unsupported settings format of \"%s\"", filename)
		}
	}

	return validateSettings(v)
}

func validateSettings(v any) error {
	validate := validator.New()
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		if tag, ok := field.Tag.Lookup("json"); ok && tag != "" {
			return strings.SplitN(tag, ",", 2)[0]
		}
		return field.Name
	})
	en := en.New()
	uni := ut.New(en, en)
	trans, _ := uni.GetTranslator("en")

	_ = validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "settings \"{0}\" required", true) // see universal-translator for details
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Namespace(), fe.StructNamespace(), fe.Field(), fe.StructField())
		return t
	})

	if err := validate.Struct(v); err != nil && trans != nil {
		if errs, ok := err.(validator.ValidationErrors); ok {
			buff := bytes.NewBufferString("")
			for _, s := range errs.Translate(trans) {
				buff.WriteString(s)
				buff.WriteString("\n")
			}
			return errors.New(buff.String())
		}
	} else {
		return err
	}

	return nil
}

func WriteSettings(v any, filename string) error {
	ext := filepath.Ext(filename)

	var j []byte
	var err error
	if strings.EqualFold(ext, ".json") ||
		strings.EqualFold(ext, ".json5") {
		j, err = json.Marshal(v)
	} else if strings.EqualFold(ext, ".yaml") ||
		strings.EqualFold(ext, ".yml") {
		j, err = yaml.Marshal(v)
	} else {
		err = errors.Errorf("the extension of file \"%s\" must be .json,.yaml,.yml", filename)
	}

	if err != nil {
		return errors.Errorf("marshal settings error: %w", err)
	}

	err = os.WriteFile(filename, j, 0o664)
	if err != nil {
		return errors.Errorf("write settings file error: %w", err)
	}

	return nil
}
