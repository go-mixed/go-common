package utils

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"go-common/utils/text"
	"io/ioutil"
	"reflect"
	"strings"
)

// LoadSettings 读取JSON格式的配置, v必须为struct的指针
// 可以传入多个文件，后面文件的配置会覆盖前面的配置
// 支持github.com/go-playground/validator的校验格式，比如：struct {Url string `json:"url" validate:"required,url,min=5,max=256"`}
func LoadSettings(v interface{}, filenames ...string) error {
	for _, filename := range filenames {
		if content, err := ioutil.ReadFile(filename); err != nil {
			return fmt.Errorf("read settings file error: %w", err)
		} else if err = text_utils.JsonUnmarshalFromBytes(content, v); err != nil {
			return fmt.Errorf("unmarshal settings file \"%s\" error: %w", filename, err)
		}
	}

	return validateSettings(v)
}

func validateSettings(v interface{}) error {
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
