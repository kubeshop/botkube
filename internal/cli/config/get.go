package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/MakeNowJust/heredoc"
	semver "github.com/hashicorp/go-version"
	"github.com/spf13/pflag"
	"go.szostok.io/version"
	"golang.org/x/exp/slices"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/install/iox"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/internal/ptr"
	"github.com/kubeshop/botkube/pkg/formatx"
)

const (
	exportJobName = "botkube-config-exporter"
	configMapName = "botkube-config-exporter"
	containerName = "botkube"
)

// ExporterOptions holds config exporter image configuration options.
type ExporterOptions struct {
	Registry   string
	Repository string
	Tag        string

	BotkubePodLabel     string
	BotkubePodNamespace string

	Timeout    time.Duration
	PollPeriod time.Duration

	CloudEnvs CloudEnvsOptions
}

type CloudEnvsOptions struct {
	EndpointEnvName string
	IDEnvName       string
	APIKeyEnvName   string
}

func (o *ExporterOptions) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&o.BotkubePodNamespace, "namespace", "n", "botkube", "Namespace of Botkube pod")
	flags.StringVarP(&o.BotkubePodLabel, "label", "l", "app=botkube", "Label used for identifying the Botkube pod")
	flags.StringVar(&o.Registry, "cfg-exporter-image-registry", "ghcr.io", "Registry for the Config Exporter job image")
	flags.StringVar(&o.Repository, "cfg-exporter-image-repo", "kubeshop/botkube-config-exporter", "Repository for the Config Exporter job image")
	flags.StringVar(&o.Tag, "cfg-exporter-image-tag", getDefaultImageTag(), "Tag of the Config Exporter job image")
	flags.DurationVar(&o.PollPeriod, "cfg-exporter-poll-period", 1*time.Second, "Interval used to check if Config Exporter job was finished")
	flags.DurationVar(&o.Timeout, "cfg-exporter-timeout", 1*time.Minute, "Maximum execution time for the Config Exporter job")

	flags.StringVar(&o.CloudEnvs.EndpointEnvName, "cloud-env-endpoint", remote.ProviderEndpointEnvKey, "Endpoint environment variable name specified under Deployment for cloud installation.")
	flags.StringVar(&o.CloudEnvs.IDEnvName, "cloud-env-id", remote.ProviderIdentifierEnvKey, "Identifier environment variable name specified under Deployment for cloud installation.")
	flags.StringVar(&o.CloudEnvs.APIKeyEnvName, "cloud-env-api-key", remote.ProviderAPIKeyEnvKey, "API key environment variable name specified under Deployment for cloud installation.")
}

func GetFromCluster(ctx context.Context, status *printer.StatusPrinter, k8sCfg *rest.Config, opts ExporterOptions, autoApprove bool) (config []byte, version string, err error) {
	defer func() {
		status.End(err == nil)
	}()
	k8sCli, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		return nil, "", fmt.Errorf("while getting k8s client: %w", err)
	}

	botkubePod, err := getBotkubePod(ctx, k8sCli, opts.BotkubePodNamespace, opts.BotkubePodLabel)
	if err != nil {
		return nil, "", fmt.Errorf("while getting botkube pod: %w", err)
	}

	botkubeContainer, found := getBotkubeContainer(botkubePod)
	if !found {
		return nil, "", fmt.Errorf("cannot find %q container in %q Pod", containerName, botkubePod.Name)
	}
	ver, err := getBotkubeVersion(botkubeContainer)
	if err != nil {
		return nil, "", fmt.Errorf("while getting botkube version: %w", err)
	}

	envs := getCloudRelatedEnvs(botkubeContainer, opts.CloudEnvs)
	if len(envs) > 1 {
		cfg, err := fetchCloudConfig(ctx, envs)
		if err != nil {
			return nil, "", fmt.Errorf("while fetching cloud configuration: %w", err)
		}
		return cfg, ver, nil
	}

	if err = createExportJob(ctx, k8sCli, botkubePod, botkubeContainer, opts, autoApprove); err != nil {
		return nil, "", fmt.Errorf("while creating config exporter job: %w", err)
	}
	status.Step("Fetching configuration")
	if err = waitForExportJob(ctx, k8sCli, opts); err != nil {
		return nil, "", fmt.Errorf("while waiting for config exporter job: %w", err)
	}

	defer cleanup(ctx, k8sCli, opts.BotkubePodNamespace)

	config, err = readConfigFromCM(ctx, k8sCli, opts.BotkubePodNamespace)
	if err != nil {
		return nil, "", fmt.Errorf("while getting exported config: %w", err)
	}

	return config, ver, nil
}

func fetchCloudConfig(ctx context.Context, envs map[string]string) ([]byte, error) {
	cfg := remote.Config{
		Endpoint:   envs[remote.ProviderEndpointEnvKey],
		Identifier: envs[remote.ProviderIdentifierEnvKey],
		APIKey:     envs[remote.ProviderAPIKeyEnvKey],
	}

	gqlClient := remote.NewDefaultGqlClient(cfg)
	deployClient := remote.NewDeploymentClient(gqlClient)
	deployConfig, err := deployClient.GetConfigWithResourceVersion(ctx)
	if err != nil {
		return nil, err
	}
	return []byte(deployConfig.YAMLConfig), nil
}

func getCloudRelatedEnvs(c corev1.Container, envs CloudEnvsOptions) map[string]string {
	exp := []string{envs.EndpointEnvName, envs.IDEnvName, envs.APIKeyEnvName}
	out := map[string]string{}
	for _, env := range c.Env {
		name := strings.TrimSpace(env.Name)
		if name == "" {
			continue
		}
		if slices.Contains(exp, name) {
			out[name] = env.Value
		}
	}
	return out
}

func getBotkubePod(ctx context.Context, k8sCli *kubernetes.Clientset, namespace, label string) (*corev1.Pod, error) {
	pods, err := k8sCli.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: label})
	if err != nil {
		return nil, err
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("there are not Pods with label %q in the %q namespace", label, namespace)
	}
	return &pods.Items[0], nil
}

func createExportJob(ctx context.Context, k8sCli *kubernetes.Clientset, botkubePod *corev1.Pod, container corev1.Container, cfg ExporterOptions, autoApprove bool) error {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      exportJobName,
			Namespace: botkubePod.Namespace,
			Labels: map[string]string{
				"app":                   exportJobName,
				"botkube.io/export-cfg": "true",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            exportJobName,
							Image:           fmt.Sprintf("%s/%s:%s", cfg.Registry, cfg.Repository, cfg.Tag),
							ImagePullPolicy: corev1.PullIfNotPresent,
							Env:             container.Env,
							VolumeMounts:    container.VolumeMounts,
						},
					},
					Volumes:            botkubePod.Spec.Volumes,
					ServiceAccountName: botkubePod.Spec.ServiceAccountName,
					RestartPolicy:      corev1.RestartPolicyNever,
				},
			},
		},
	}

	_, err := k8sCli.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil && apierrors.IsAlreadyExists(err) {
		can, err := checkIfCanDelete(autoApprove)
		if err != nil {
			return fmt.Errorf("while checking if can delete: %w", err)
		}

		if can {
			_ = k8sCli.BatchV1().Jobs(job.Namespace).Delete(ctx, exportJobName, metav1.DeleteOptions{
				GracePeriodSeconds: ptr.FromType[int64](0),
				PropagationPolicy:  ptr.FromType(metav1.DeletePropagationOrphan),
			})
			err := wait.PollUntilContextTimeout(ctx, time.Second, cfg.Timeout, false, func(ctx context.Context) (done bool, err error) {
				_, err = k8sCli.BatchV1().Jobs(job.Namespace).Get(ctx, job.Namespace, metav1.GetOptions{})
				if apierrors.IsNotFound(err) {
					return true, nil
				}
				return false, err
			})
			if err != nil {
				return fmt.Errorf("while waiting for job deletion: %w", err)
			}

			_, err = k8sCli.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("while re-creating job: %w", err)
			}
			return nil
		}
	}
	return err
}

func checkIfCanDelete(autoApprove bool) (bool, error) {
	if autoApprove {
		return true, nil
	}

	run := false
	prompt := &survey.Confirm{
		Message: "The migration job already exist. Do you want to re-create it?",
		Default: true,
	}

	// we use os.Stderr as user may user piping output into file, so we don't want to break it.
	questionIndent := iox.NewIndentFileWriter(os.Stderr, "?", 1) // we indent questions by 1 space to match the step layout
	err := survey.AskOne(prompt, &run, survey.WithStdio(os.Stdin, questionIndent, os.Stderr))
	return run, err
}

func getBotkubeContainer(botkubePod *corev1.Pod) (corev1.Container, bool) {
	for _, c := range botkubePod.Spec.Containers {
		if c.Name == containerName {
			return c, true
		}
	}
	return corev1.Container{}, false
}

func waitForExportJob(ctx context.Context, k8sCli *kubernetes.Clientset, opts ExporterOptions) error {
	ctxWithTimeout, cancelFn := context.WithTimeout(ctx, opts.Timeout)
	defer cancelFn()

	ticker := time.NewTicker(opts.PollPeriod)
	defer ticker.Stop()

	var job *batchv1.Job
	for {
		select {
		case <-ctxWithTimeout.Done():
			var errMsg string
			if errors.Is(ctxWithTimeout.Err(), context.Canceled) {
				errMsg = "Interrupted while waiting for export config job."
			} else {
				errMsg = fmt.Sprintf("Timeout (%s) occurred while waiting for export config job.", opts.Timeout.String())
			}

			if job != nil {
				errMsg = heredoc.Docf(`
				%s

				To get all Botkube logs, run:
				
				  kubectl logs -n %s jobs/%s
				`, errMsg, job.Namespace, job.Name)
			}

			if cli.VerboseMode.IsEnabled() && job != nil {
				job.ManagedFields = nil
				errMsg = fmt.Sprintf("%s\n\nDEBUG:\nJob definition:\n\n%s", errMsg, formatx.StructDumper().Sdump(job))
			}

			return errors.New(errMsg)
		case <-ticker.C:
			var err error
			job, err = k8sCli.BatchV1().Jobs(opts.BotkubePodNamespace).Get(ctx, exportJobName, metav1.GetOptions{})
			if err != nil {
				fmt.Println("Error getting export config job: ", err.Error())
				continue
			}

			if job.Status.Succeeded > 0 {
				return nil
			}
		}
	}
}

func readConfigFromCM(ctx context.Context, k8sCli *kubernetes.Clientset, namespace string) ([]byte, error) {
	configMap, err := k8sCli.CoreV1().ConfigMaps(namespace).Get(ctx, configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	oldApproachStorage := configMap.BinaryData["config.yaml"]
	if len(oldApproachStorage) > 0 {
		return oldApproachStorage, nil
	}

	return []byte(configMap.Data["config.yaml"]), nil
}

func cleanup(ctx context.Context, k8sCli *kubernetes.Clientset, namespace string) {
	foreground := metav1.DeletePropagationForeground
	_ = k8sCli.BatchV1().Jobs(namespace).Delete(ctx, exportJobName, metav1.DeleteOptions{PropagationPolicy: &foreground})
	_ = k8sCli.CoreV1().ConfigMaps(namespace).Delete(ctx, configMapName, metav1.DeleteOptions{})
}

func getBotkubeVersion(c corev1.Container) (string, error) {
	fqin := strings.Split(c.Image, ":")
	if len(fqin) > 1 {
		return fqin[len(fqin)-1], nil
	}
	return "", errors.New("unable to get botkube version")
}

func getDefaultImageTag() string {
	imageTag := "v9.99.9-dev"
	ver, err := semver.NewSemver(version.Get().Version)
	if err == nil {
		imageTag = "v" + ver.String()
	}
	return imageTag
}
