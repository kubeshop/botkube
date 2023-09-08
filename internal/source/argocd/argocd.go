package argocd

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

var _ source.Source = (*Source)(nil)

var (
	//go:embed config-jsonschema.json
	configJSONSchema string

	//go:embed req-jsonschema.json
	requestJSONSchema string

	//go:embed default-config.yaml
	defaultConfigYAML string
)

const (
	// PluginName is the name of the source plugin.
	PluginName = "argocd"

	description = "Argo source plugin is used to get ArgoCD trigger-based notifications."
)

// Source defines ArgoCD source plugin.
type Source struct {
	pluginVersion string
	log           logrus.FieldLogger
	cfg           Config
	srcCtx        source.CommonSourceContext
}

// NewSource returns a new instance of Source.
func NewSource(version string) *Source {
	return &Source{
		pluginVersion: version,
	}
}

type subscription struct {
	TriggerName string
	WebhookName string
	Application config.K8sResourceRef
}

// Stream set-ups ArgoCD notifications.
func (s *Source) Stream(ctx context.Context, input source.StreamInput) (source.StreamOutput, error) {
	if err := pluginx.ValidateKubeConfigProvided(PluginName, input.Context.KubeConfig); err != nil {
		return source.StreamOutput{}, err
	}

	cfg, err := mergeConfigs(input.Configs)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}
	s.cfg = cfg
	s.log = loggerx.New(cfg.Log)

	s.srcCtx = input.Context.CommonSourceContext

	k8sCli, err := s.getK8sClient(input.Context.KubeConfig)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while creating K8s clientset: %w", err)
	}

	s.log.Info("Preparing configuration...")

	err = retry.Do(
		func() error {
			return s.setupArgoNotifications(ctx, k8sCli)
		},
		retry.OnRetry(func(n uint, err error) {
			s.log.Errorf("")
		}),
		retry.DelayType(retry.RandomDelay), // Randomize the retry time as ConfigMap is updated and there might be conflicts when there are multiple plugin configurations
		retry.MaxJitter(5*time.Second),
		retry.Attempts(5),
		retry.LastErrorOnly(true),
	)
	if err != nil {
		return source.StreamOutput{}, fmt.Errorf("while configuring Argo notifications: %w", err)
	}

	s.log.Infof("Setup successful for source configuration %q", input.Context.SourceName)
	return source.StreamOutput{}, nil
}

// HandleExternalRequest handles external requests from ArgoCD.
func (s *Source) HandleExternalRequest(_ context.Context, input source.ExternalRequestInput) (source.ExternalRequestOutput, error) {
	payload := formatx.StructDumper().Sdump(string(input.Payload))
	s.log.WithField("payload", payload).Debug("Handling external request...")
	fallbackTimestamp := time.Now()

	var reqBody IncomingRequestBody
	err := json.Unmarshal(input.Payload, &reqBody)
	if err != nil {
		return source.ExternalRequestOutput{}, fmt.Errorf("while unmarshalling payload: %w", err)
	}

	msg := reqBody.Message
	if msg.Timestamp.IsZero() {
		msg.Timestamp = fallbackTimestamp
	}

	if input.Context.IsInteractivitySupported {
		section := s.generateInteractivitySection(reqBody)
		if section != nil {
			msg.Sections = append(msg.Sections, *section)
		}
	} else {
		msg.Type = api.NonInteractiveSingleSection
		lastSectionIdx := len(msg.Sections) - 1
		if lastSectionIdx != -1 {
			msg.Sections[lastSectionIdx].TextFields = append(msg.Sections[lastSectionIdx].TextFields, s.generateNonInteractiveFields(reqBody)...)
		}
	}

	return source.ExternalRequestOutput{
		Event: source.Event{
			Message:   msg,
			RawObject: reqBody,
		},
	}, nil
}

// Metadata returns metadata of the ArgoCD configuration.
func (s *Source) Metadata(_ context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     s.pluginVersion,
		Description: description,
		JSONSchema: api.JSONSchema{
			Value: configJSONSchema,
		},
		ExternalRequest: api.ExternalRequestMetadata{
			Payload: api.ExternalRequestPayload{
				JSONSchema: api.JSONSchema{
					Value: requestJSONSchema,
				},
			},
		},
	}, nil
}

func (s *Source) getK8sClient(k8sBytes []byte) (*dynamic.DynamicClient, error) {
	kubeConfig, err := clientcmd.RESTConfigFromKubeConfig(k8sBytes)
	if err != nil {
		return nil, fmt.Errorf("while reading kube config: %v", err)
	}

	dynamicK8sCli, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("while creating dynamic K8s client: %w", err)
	}

	return dynamicK8sCli, nil
}
