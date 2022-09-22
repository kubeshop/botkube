package config

import (
	"fmt"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/hashicorp/go-multierror"

	multierrx "github.com/kubeshop/botkube/pkg/multierror"
)

const nsIncludeTag = "ns-include-regex"

var warnsOnlyTags = map[string]struct{}{
	nsIncludeTag: {},
}

// ValidateResult holds the validation results.
type ValidateResult struct {
	Criticals *multierror.Error
	Warnings  *multierror.Error
}

// ValidateStruct validates a given struct based on the `validate` field tag.
func ValidateStruct(in any) (ValidateResult, error) {
	validate := validator.New()

	trans := ut.New(en.New()).GetFallback() // Currently we don't support other that en translations.
	if err := en_translations.RegisterDefaultTranslations(validate, trans); err != nil {
		return ValidateResult{}, err
	}

	if err := registerNamespaceValidator(validate, trans); err != nil {
		return ValidateResult{}, err
	}

	err := validate.Struct(in)
	if err == nil {
		return ValidateResult{}, nil
	}

	errs, ok := err.(validator.ValidationErrors)
	if !ok {
		return ValidateResult{}, err
	}
	result := ValidateResult{
		Criticals: multierrx.New(),
		Warnings:  multierrx.New(),
	}

	for _, e := range errs {
		msg := fmt.Errorf("Key: '%s' %s", e.StructNamespace(), e.Translate(trans))

		if _, found := warnsOnlyTags[e.Tag()]; found {
			result.Warnings = multierrx.Append(result.Warnings, msg)
			continue
		}

		result.Criticals = multierrx.Append(result.Criticals, msg)
	}
	return result, nil
}

func registerNamespaceValidator(validate *validator.Validate, trans ut.Translator) error {
	// NOTE: only have to register a non-pointer type for 'Namespaces', validator
	// internally dereferences it.
	validate.RegisterStructValidation(namespacesStructValidator, Namespaces{})

	registerFn := func(ut ut.Translator) error {
		return ut.Add(nsIncludeTag, "{0} matches both all and exact namespaces", false)
	}

	return validate.RegisterTranslation(nsIncludeTag, trans, registerFn, translateFunc)
}

func namespacesStructValidator(sl validator.StructLevel) {
	ns, ok := sl.Current().Interface().(Namespaces)
	if !ok {
		return
	}

	if len(ns.Include) < 2 {
		return
	}

	foundAllNamespaceIndicator := func() bool {
		for _, name := range ns.Include {
			if name == AllNamespaceIndicator {
				return true
			}
		}
		return false
	}

	if foundAllNamespaceIndicator() {
		sl.ReportError(ns.Include, "Include", "Include", nsIncludeTag, "")
	}
}

// copied from: https://github.com/go-playground/validator/blob/9e2ea4038020b5c7e3802a21cfa4e3afcfdcd276/translations/en/en.go#L1391-L1399
func translateFunc(ut ut.Translator, fe validator.FieldError) string {
	t, err := ut.T(fe.Tag(), fe.Field())
	if err != nil {
		return fe.(error).Error()
	}

	return t
}
