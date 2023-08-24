package config

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	semver "github.com/hashicorp/go-version"
	"github.com/spf13/pflag"
	"go.szostok.io/version"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kubeshop/botkube/internal/cli"
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
}

func (o *ExporterOptions) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&o.BotkubePodNamespace, "namespace", "n", "botkube", "Namespace of Botkube pod")
	flags.StringVarP(&o.BotkubePodLabel, "label", "l", "app=botkube", "Label used for identifying the Botkube pod")
	flags.StringVar(&o.Registry, "cfg-exporter-image-registry", "ghcr.io", "Registry for the Config Exporter job image")
	flags.StringVar(&o.Repository, "cfg-exporter-image-repo", "kubeshop/botkube-config-exporter", "Repository for the Config Exporter job image")
	flags.StringVar(&o.Tag, "cfg-exporter-image-tag", getDefaultImageTag(), "Tag of the Config Exporter job image")
	flags.DurationVar(&o.PollPeriod, "cfg-exporter-poll-period", 1*time.Second, "Interval used to check if Config Exporter job was finished")
	flags.DurationVar(&o.Timeout, "cfg-exporter-timeout", 1*time.Minute, "Maximum execution time for the Config Exporter job")
}

func GetFromCluster(ctx context.Context, k8sCfg *rest.Config, opts ExporterOptions) ([]byte, string, error) {
	k8sCli, err := kubernetes.NewForConfig(k8sCfg)
	if err != nil {
		return nil, "", fmt.Errorf("while getting k8s client: %w", err)
	}
	defer cleanup(ctx, k8sCli, opts.BotkubePodNamespace)

	botkubePod, err := getBotkubePod(ctx, k8sCli, opts.BotkubePodNamespace, opts.BotkubePodLabel)
	if err != nil {
		return nil, "", fmt.Errorf("while getting botkube pod: %w", err)
	}

	if err = createExportJob(ctx, k8sCli, botkubePod, opts); err != nil {
		return nil, "", fmt.Errorf("while creating config exporter job: %w", err)
	}

	if err = waitForExportJob(ctx, k8sCli, opts); err != nil {
		return nil, "", fmt.Errorf("while waiting for config exporter job: %w", err)
	}

	config, err := readConfigFromCM(ctx, k8sCli, opts.BotkubePodNamespace)
	if err != nil {
		return nil, "", fmt.Errorf("while getting exported config: %w", err)
	}

	ver, err := getBotkubeVersion(botkubePod)
	if err != nil {
		return nil, "", fmt.Errorf("while getting botkube version: %w", err)
	}
	return config, ver, nil
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

func createExportJob(ctx context.Context, k8sCli *kubernetes.Clientset, botkubePod *corev1.Pod, cfg ExporterOptions) error {
	var container corev1.Container
	for _, c := range botkubePod.Spec.Containers {
		if c.Name == "botkube" {
			container = c
			break
		}
	}

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

	_, err := k8sCli.BatchV1().Jobs(botkubePod.Namespace).Create(ctx, job, metav1.CreateOptions{})

	return err
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

			errMsg := fmt.Sprintf("export config job failed: %s", context.Canceled.Error())

			if cli.VerboseMode.IsEnabled() && job != nil {
				job.ManagedFields = nil
				errMsg = fmt.Sprintf("%s\n\nDEBUG:\nJob definition:\n\n%s", errMsg, formatx.StructDumper().Sdump(job))
			}

			// TODO: Add ability to keep the job if it fails and improve the error
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

func getBotkubeVersion(p *corev1.Pod) (string, error) {
	for _, c := range p.Spec.Containers {
		if c.Name == containerName {
			fqin := strings.Split(c.Image, ":")
			if len(fqin) > 1 {
				return fqin[len(fqin)-1], nil
			}
			break
		}
	}
	return "", fmt.Errorf("unable to get botkube version: pod %q does not have botkube container", p.Name)
}

func getDefaultImageTag() string {
	imageTag := "v9.99.9-dev"
	ver, err := semver.NewSemver(version.Get().Version)
	if err == nil {
		imageTag = "v" + ver.String()
	}
	return imageTag
}
