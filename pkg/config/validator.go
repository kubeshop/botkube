package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
	"github.com/hashicorp/go-multierror"

	multierrx "github.com/kubeshop/botkube/pkg/multierror"
)

const (
	nsIncludeTag   = "ns-include-regex"
	appTokenPrefix = "xapp"
	botTokenPrefix = "xoxb"
)

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

	if err := registerCustomTranslations(validate, trans); err != nil {
		return ValidateResult{}, err
	}

	if err := registerNamespaceValidator(validate, trans); err != nil {
		return ValidateResult{}, err
	}

	validate.RegisterStructValidation(slackStructTokenValidator, Slack{})

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

func registerCustomTranslations(validate *validator.Validate, trans ut.Translator) error {
	excludedWith := func(ut ut.Translator) error {
		return ut.Add("excluded_with", "{0} and {1} fields are mutually exclusive", false)
	}

	if err := validate.RegisterTranslation("excluded_with", trans, excludedWith, translateFunc); err != nil {
		return err
	}

	requiredWith := func(ut ut.Translator) error {
		return ut.Add("required_with", "{0} and {1} must be specified together", false)
	}

	if err := validate.RegisterTranslation("required_with", trans, requiredWith, translateFunc); err != nil {
		return err
	}

	startsWith := func(ut ut.Translator) error {
		return ut.Add("startswith", "{0} must have the prefix {1}", false)
	}

	if err := validate.RegisterTranslation("startswith", trans, startsWith, translateFunc); err != nil {
		return err
	}

	return nil
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

func slackStructTokenValidator(sl validator.StructLevel) {
	slack, ok := sl.Current().Interface().(Slack)

	if !ok || !slack.Enabled {
		return
	}

	if slack.Token != "" {
		if slack.BotToken != "" {
			sl.ReportError(slack.BotToken, "BotToken", "BotToken", "excluded_with", "Token")
		}
		if slack.AppToken != "" {
			sl.ReportError(slack.AppToken, "AppToken", "AppToken", "excluded_with", "Token")
		}
		if !strings.HasPrefix(slack.Token, botTokenPrefix) {
			sl.ReportError(slack.Token, "Token", "Token", "startswith", botTokenPrefix)
		}
	}

	if slack.BotToken != "" && slack.AppToken == "" {
		sl.ReportError(slack.AppToken, "AppToken", "AppToken", "required_with", "BotToken")

		if !strings.HasPrefix(slack.BotToken, botTokenPrefix) {
			sl.ReportError(slack.BotToken, "BotToken", "BotToken", "startswith", botTokenPrefix)
		}
	}

	if slack.AppToken != "" && slack.BotToken == "" {
		sl.ReportError(slack.BotToken, "BotToken", "BotToken", "required_with", "AppToken")

		if !strings.HasPrefix(slack.AppToken, appTokenPrefix) {
			sl.ReportError(slack.AppToken, "AppToken", "AppToken", "startswith", appTokenPrefix)
		}
	}

	if slack.Token == "" && slack.BotToken == "" && slack.AppToken == "" {
		sl.ReportError(slack.Token, "Token", "Token", "required", "")
	}
}

// copied from: https://github.com/go-playground/validator/blob/9e2ea4038020b5c7e3802a21cfa4e3afcfdcd276/translations/en/en.go#L1391-L1399
func translateFunc(ut ut.Translator, fe validator.FieldError) string {
	t, err := ut.T(fe.Tag(), fe.Field(), fe.Param())
	if err != nil {
		return fe.(error).Error()
	}

	return t
}
