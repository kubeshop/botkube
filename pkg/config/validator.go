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
	nsIncludeTag      = "ns-include-regex"
	invalidBindingTag = "invalid_binding"
	appTokenPrefix    = "xapp-"
	botTokenPrefix    = "xoxb-"
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
	if err := registerBindingsValidator(validate, trans); err != nil {
		return ValidateResult{}, err
	}

	validate.RegisterStructValidation(slackStructTokenValidator, Slack{})
	validate.RegisterStructValidation(socketSlackStructTokenValidator, SocketSlack{})

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

func registerCustomTranslations(validate *validator.Validate, trans ut.Translator) error {
	startsWith := func(ut ut.Translator) error {
		return ut.Add("invalid_slack_token", "{0} {1}", false)
	}

	if err := validate.RegisterTranslation("invalid_slack_token", trans, startsWith, translateFunc); err != nil {
		return err
	}

	return nil
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

func registerBindingsValidator(validate *validator.Validate, trans ut.Translator) error {
	validate.RegisterStructValidation(bindingsStructValidator, Config{})

	registerFn := func(ut ut.Translator) error {
		return ut.Add(invalidBindingTag, "{0} {1}", false)
	}

	return validate.RegisterTranslation(invalidBindingTag, trans, registerFn, translateFunc)
}

func slackStructTokenValidator(sl validator.StructLevel) {
	slack, ok := sl.Current().Interface().(Slack)

	if !ok || !slack.Enabled {
		return
	}

	if slack.Token == "" {
		sl.ReportError(slack.Token, "Token", "Token", "required", "")
		return
	}

	if !strings.HasPrefix(slack.Token, botTokenPrefix) {
		msg := fmt.Sprintf("must have the %s prefix. Learn more at https://botkube.io/docs/installation/slack/#install-botkube-slack-app-to-your-slack-workspace", botTokenPrefix)
		sl.ReportError(slack.Token, "Token", "Token", "invalid_slack_token", msg)
	}
}

func socketSlackStructTokenValidator(sl validator.StructLevel) {
	slack, ok := sl.Current().Interface().(SocketSlack)

	if !ok || !slack.Enabled {
		return
	}

	if slack.AppToken == "" {
		sl.ReportError(slack.AppToken, "AppToken", "AppToken", "required", "")
	}

	if slack.BotToken == "" {
		sl.ReportError(slack.BotToken, "BotToken", "BotToken", "required", "")
	}

	if !strings.HasPrefix(slack.BotToken, botTokenPrefix) {
		msg := fmt.Sprintf("must have the %s prefix. Learn more at https://botkube.io/docs/installation/socketslack/#obtain-bot-token", botTokenPrefix)
		sl.ReportError(slack.BotToken, "BotToken", "BotToken", "invalid_slack_token", msg)
	}

	if !strings.HasPrefix(slack.AppToken, appTokenPrefix) {
		msg := fmt.Sprintf("must have the %s prefix. Learn more at https://botkube.io/docs/installation/socketslack/#generate-and-obtain-app-level-token", appTokenPrefix)
		sl.ReportError(slack.AppToken, "AppToken", "AppToken", "invalid_slack_token", msg)
	}
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

func bindingsStructValidator(sl validator.StructLevel) {
	conf, ok := sl.Current().Interface().(Config)
	if !ok {
		return
	}
	for _, comm := range conf.Communications {
		// slack
		for _, ch := range comm.Slack.Channels {
			for _, source := range ch.Bindings.Sources {
				if _, ok := conf.Sources[source]; !ok {
					sl.ReportError(source, source, source, invalidBindingTag, fmt.Sprintf("slack binding not defined: %s", source))
				}
			}
			for _, executor := range ch.Bindings.Executors {
				if _, ok := conf.Executors[executor]; !ok {
					sl.ReportError(executor, executor, executor, invalidBindingTag, fmt.Sprintf("slack binding not defined: %s", executor))
				}
			}
		}
		// socketSlack
		for _, ch := range comm.SocketSlack.Channels {
			for _, source := range ch.Bindings.Sources {
				if _, ok := conf.Sources[source]; !ok {
					sl.ReportError(source, source, source, invalidBindingTag, fmt.Sprintf("slack binding not defined: %s", source))
				}
			}
			for _, executor := range ch.Bindings.Executors {
				if _, ok := conf.Executors[executor]; !ok {
					sl.ReportError(executor, executor, executor, invalidBindingTag, fmt.Sprintf("slack binding not defined: %s", executor))
				}
			}
		}
		// mattermost
		for _, ch := range comm.Mattermost.Channels {
			for _, source := range ch.Bindings.Sources {
				if _, ok := conf.Sources[source]; !ok {
					sl.ReportError(source, source, source, invalidBindingTag, fmt.Sprintf("mattermost binding not defined: %s", source))
				}
			}
			for _, executor := range ch.Bindings.Executors {
				if _, ok := conf.Executors[executor]; !ok {
					sl.ReportError(executor, executor, executor, invalidBindingTag, fmt.Sprintf("mattermost binding not defined: %s", executor))
				}
			}
		}
		// discord
		for _, ch := range comm.Discord.Channels {
			for _, source := range ch.Bindings.Sources {
				if _, ok := conf.Sources[source]; !ok {
					sl.ReportError(source, source, source, invalidBindingTag, fmt.Sprintf("discord binding not defined: %s", source))
				}
			}
			for _, executor := range ch.Bindings.Executors {
				if _, ok := conf.Executors[executor]; !ok {
					sl.ReportError(executor, executor, executor, invalidBindingTag, fmt.Sprintf("discord binding not defined: %s", executor))
				}
			}
		}
		// teams
		for _, source := range comm.Teams.Bindings.Sources {
			if _, ok := conf.Sources[source]; !ok {
				sl.ReportError(source, source, source, invalidBindingTag, fmt.Sprintf("teams binding not defined: %s", source))
			}
		}
		for _, executor := range comm.Teams.Bindings.Executors {
			if _, ok := conf.Executors[executor]; !ok {
				sl.ReportError(executor, executor, executor, invalidBindingTag, fmt.Sprintf("teams binding not defined: %s", executor))
			}
		}
		// webhook
		for _, source := range comm.Webhook.Bindings.Sources {
			if _, ok := conf.Sources[source]; !ok {
				sl.ReportError(source, source, source, invalidBindingTag, fmt.Sprintf("discord binding not defined: %s", source))
			}
		}
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
