package main

import (
	"context"

	cfginternal "github.com/kubeshop/botkube/internal/config"
	"github.com/kubeshop/botkube/pkg/config"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlutil "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func main() {
	files, _, err := cfginternal.NewEnvProvider().Configs(context.Background())
	if err != nil {
		panic(err)
	}
	conf, _, err := config.LoadWithDefaults(files)
	if err != nil {
		panic(err)
	}
	yamlBytes, err := yaml.Marshal(conf)
	if err != nil {
		panic(err)
	}
	if err := createOrUpdateCM(context.Background(), yamlBytes); err != nil {
		panic(err)
	}

}

func createOrUpdateCM(ctx context.Context, config []byte) error {
	k8sClient, err := newK8sClient()
	if err != nil {
		return err
	}
	cm := newCM()
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		_, corErr := ctrlutil.CreateOrUpdate(ctx, k8sClient, cm, func() error {
			cm.BinaryData = map[string][]byte{
				"config.yaml": config,
			}
			return nil
		})
		return corErr
	})
}

func newK8sClient() (client.Client, error) {
	k8sConfig, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, err
	}
	return client.New(k8sConfig, client.Options{})
}

func newCM() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "botkube-migration",
			Namespace: "botkube",
			Labels: map[string]string{
				"app": "botkube-migration",
			},
		},
	}
}
