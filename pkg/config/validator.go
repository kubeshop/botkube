package config

import (
	"github.com/go-playground/validator/v10"

	"github.com/kubeshop/botkube/pkg/multierror"
)

// ValidateStruct validates a given struct based on the `validate` field tag.
func ValidateStruct(in any) error {
	validate := validator.New()
	err := validate.Struct(in)
	if err != nil {
		errs, ok := err.(validator.ValidationErrors)
		if !ok {
			return err
		}

		result := multierror.New()
		for _, e := range errs {
			result = multierror.Append(result, e)
		}

		return result.ErrorOrNil()
	}

	return nil
}
