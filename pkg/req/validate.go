package req

import (
	"github.com/go-playground/validator/v10"
)

func IsValid[T any](payload T) error {
	valid := validator.New()
	err := valid.Struct(payload)
	if err != nil {
		return err
	}
	return nil
}
