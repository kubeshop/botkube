package argocd

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"strings"
	gotemplate "text/template"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api/source"
)

const (
	goTplOpeningTag = "{{"
)

// renderTemplateName renders template name.
// In fact, ConfigMap keys can contain slashes, so we don't need to do it for templates
// but hey, let's keep the same normalization rules across the plugin codebase.
func (s *Source) renderTemplateName(tpl string, srcCtx source.CommonSourceContext) (string, error) {
	s.log.Debugf("Rendering template name %q...", tpl)
	templateName, err := renderStringIfTemplate(tpl, srcCtx)
	if err != nil {
		return "", fmt.Errorf("while rendering template name: %w", err)
	}
	return normalize(s.log, templateName, maxTemplateNameLength), nil
}

func (s *Source) renderTriggerName(tpl string, srcCtx source.CommonSourceContext) (string, error) {
	s.log.Debugf("Rendering trigger name %q...", tpl)
	triggerName, err := renderStringIfTemplate(tpl, srcCtx)
	if err != nil {
		return "", fmt.Errorf("while rendering trigger name: %w", err)
	}
	return normalize(s.log, triggerName, maxTriggerNameLength), nil
}

func (s *Source) renderWebhookName(tpl string, srcCtx source.CommonSourceContext) (string, error) {
	s.log.Debugf("Rendering webhook name %q...", tpl)
	webhookName, err := renderStringIfTemplate(tpl, srcCtx)
	if err != nil {
		return "", fmt.Errorf("while rendering webhook name: %w", err)
	}
	return normalize(s.log, webhookName, maxWebhookNameLength), nil
}

func renderStringIfTemplate(tpl string, srcCtx source.CommonSourceContext) (string, error) {
	if !strings.Contains(tpl, goTplOpeningTag) {
		return tpl, nil
	}

	tmpl, err := gotemplate.New("tpl").Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("while parsing template %q: %w", tpl, err)
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, srcCtx)
	if err != nil {
		return "", fmt.Errorf("while rendering string %q: %w", tpl, err)
	}

	return result.String(), nil
}

func normalize(log logrus.FieldLogger, in string, maxSize int) string {
	out := in
	defer log.Debugf("Normalized %q to %q", in, out)

	// replace all special characters with `-`
	out = allowedCharsRegex.ReplaceAllString(out, "-")

	// make it lowercase
	out = strings.ToLower(out)

	if len(out) <= maxSize {
		return out
	}

	// nolint:gosec // false positive
	h := sha1.New()
	h.Write([]byte(in))
	hash := hex.EncodeToString(h.Sum(nil))

	hashMaxSize := maxSize - 2 // 2 chars for the `b-` prefix
	// if the hash is too long, truncate it
	if len(hash) > hashMaxSize {
		hash = hash[:hashMaxSize]
	}

	return fmt.Sprintf("%s-%s", namePrefix, hash)
}
