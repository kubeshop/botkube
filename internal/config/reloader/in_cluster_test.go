package reloader

import (
	"context"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/scheme"
	clienttesting "k8s.io/client-go/testing"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/pkg/config"
)

var (
	namespace  = "test-ns"
	namespace2 = "diff-ns"

	timeout = 3 * time.Second
)

func TestInClusterConfigReloader_Do(t *testing.T) {
	// given
	initialResources := fixResources()
	cfg := config.CfgWatcher{
		InCluster: config.InClusterCfgWatcher{
			InformerResyncPeriod: 0,
		},
		Deployment: config.K8sResourceRef{
			Namespace: namespace,
		},
	}

	testCases := []struct {
		Name            string
		Operation       func(ctx context.Context, t *testing.T, cli dynamic.Interface)
		ExpectedRestart bool
	}{
		{
			Name:            "No operation",
			Operation:       nil,
			ExpectedRestart: false,
		},

		// create
		{
			Name: "Create arbitrary ConfigMap",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(configMapGVR).Namespace(namespace).Create(ctx, fixConfigMap(t, "cm2", namespace, false, false), metav1.CreateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Create labeled ConfigMap in different namespace",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(configMapGVR).Namespace(namespace2).Create(ctx, fixConfigMap(t, "cm2", namespace2, true, false), metav1.CreateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Create labeled ConfigMap",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(configMapGVR).Namespace(namespace).Create(ctx, fixConfigMap(t, "cm2", namespace, true, false), metav1.CreateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: true,
		},
		{
			Name: "Create arbitrary Secret",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(secretGVR).Namespace(namespace).Create(ctx, fixSecret(t, "secret2", namespace, false, false), metav1.CreateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Create labeled Secret in different namespace",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(secretGVR).Namespace(namespace2).Create(ctx, fixSecret(t, "secret2", namespace2, true, false), metav1.CreateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Create labeled Secret",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(secretGVR).Namespace(namespace).Create(ctx, fixSecret(t, "secret2", namespace, true, false), metav1.CreateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: true,
		},

		// update
		{
			Name: "Update arbitrary ConfigMap",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(configMapGVR).Namespace(namespace).Update(ctx, fixConfigMap(t, "cm", namespace, false, true), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Update labeled ConfigMap in different namespace",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(configMapGVR).Namespace(namespace2).Update(ctx, fixConfigMap(t, "cm-labeled-diff-ns", namespace2, true, true), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Update labeled ConfigMap with new data",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(configMapGVR).Namespace(namespace).Update(ctx, fixConfigMap(t, "cm-labeled", namespace, true, true), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: true,
		},
		{
			Name: "Update labeled ConfigMap with old data",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(configMapGVR).Namespace(namespace).Update(ctx, fixConfigMap(t, "cm-labeled", namespace, true, false), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Update arbitrary Secret",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(secretGVR).Namespace(namespace).Update(ctx, fixSecret(t, "secret", namespace, false, true), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Update labeled Secret in different namespace",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(secretGVR).Namespace(namespace2).Update(ctx, fixSecret(t, "secret-labeled-diff-ns", namespace2, true, true), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Update labeled Secret with new data",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(secretGVR).Namespace(namespace).Update(ctx, fixSecret(t, "secret-labeled", namespace, true, true), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: true,
		},
		{
			Name: "Update labeled Secret with old data",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				_, err := cli.Resource(secretGVR).Namespace(namespace).Update(ctx, fixSecret(t, "secret-labeled", namespace, true, false), metav1.UpdateOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},

		// delete
		{
			Name: "Delete arbitrary ConfigMap",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				err := cli.Resource(configMapGVR).Namespace(namespace).Delete(ctx, "cm", metav1.DeleteOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Delete labeled ConfigMap in different namespace",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				err := cli.Resource(configMapGVR).Namespace(namespace2).Delete(ctx, "cm-labeled-diff-ns", metav1.DeleteOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Delete labeled ConfigMap",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				err := cli.Resource(configMapGVR).Namespace(namespace).Delete(ctx, "cm-labeled", metav1.DeleteOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: true,
		},
		{
			Name: "Delete arbitrary Secret",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				err := cli.Resource(secretGVR).Namespace(namespace).Delete(ctx, "secret", metav1.DeleteOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Delete labeled Secret in different namespace",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				err := cli.Resource(secretGVR).Namespace(namespace2).Delete(ctx, "secret-labeled-diff-ns", metav1.DeleteOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: false,
		},
		{
			Name: "Delete labeled Secret",
			Operation: func(ctx context.Context, t *testing.T, cli dynamic.Interface) {
				err := cli.Resource(secretGVR).Namespace(namespace).Delete(ctx, "secret-labeled", metav1.DeleteOptions{})
				require.NoError(t, err)
			},
			ExpectedRestart: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			dynamicCli, waitForGVRsWatchFn := createFakeK8sCli(initialResources...)
			restarter := &noopRestarter{}
			reloader, err := NewInClusterConfigReloader(
				loggerx.NewNoop(),
				dynamicCli,
				cfg,
				restarter,
				analytics.NewNoopReporter(),
			)
			require.NoError(t, err)

			ctx, cancelFn := context.WithTimeout(context.Background(), timeout)
			defer cancelFn()
			// when
			wg := sync.WaitGroup{}
			var rErr error
			wg.Add(1)
			go func(ctx context.Context) {
				defer wg.Done()
				rErr = reloader.Do(ctx)
			}(ctx)

			reloader.InformerFactory().WaitForCacheSync(ctx.Done())
			waitForGVRsWatchFn(t, ctx)

			// then
			require.False(t, restarter.restarted)

			// when
			if tc.Operation != nil {
				tc.Operation(ctx, t, dynamicCli)
				time.Sleep(50 * time.Millisecond) // make sure the operation can be processed before canceling the context
			}

			cancelFn()
			wg.Wait()

			// then
			require.NoError(t, rErr)
			assert.Equal(t, tc.ExpectedRestart, restarter.restarted)
		})
	}
}

type noopRestarter struct {
	restarted bool
}

func (n *noopRestarter) Do(_ context.Context) error {
	n.restarted = true
	return nil
}

func fixResources() []runtime.Object {
	return []runtime.Object{
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm",
				Namespace: namespace,
			},
			Data: map[string]string{
				"test": "test",
			},
		},
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm-labeled",
				Namespace: namespace,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			Data: map[string]string{
				"test": "test",
			},
		},
		&v1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "cm-labeled-diff-ns",
				Namespace: namespace2,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			Data: map[string]string{
				"test": "test",
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret",
				Namespace: namespace,
			},
			Data: map[string][]byte{
				"test": []byte("test"),
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-labeled",
				Namespace: namespace,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			Data: map[string][]byte{
				"test": []byte("test"),
			},
		},
		&v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-labeled-diff-ns",
				Namespace: namespace2,
				Labels: map[string]string{
					labelKey: labelValue,
				},
			},
			Data: map[string][]byte{
				"test": []byte("test"),
			},
		},
	}
}

func fixConfigMap(t *testing.T, name string, namespace string, labeled, isNewObj bool) *unstructured.Unstructured {
	cm := &v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "1",
			Name:            name,
			Namespace:       namespace,
		},
		Data: map[string]string{
			"test": "test",
		},
	}
	if isNewObj {
		cm.ResourceVersion = "2"
		cm.Data = map[string]string{
			"new": "data",
		}
	}

	if labeled {
		cm.SetLabels(map[string]string{
			labelKey: labelValue,
		})
	}

	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cm)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: unstrObj}
}

func fixSecret(t *testing.T, name string, namespace string, labeled, isNewObj bool) *unstructured.Unstructured {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			ResourceVersion: "1",
			Name:            name,
			Namespace:       namespace,
		},
		Data: map[string][]byte{
			"test": []byte("test"),
		},
	}
	if isNewObj {
		secret.ResourceVersion = "2"
		secret.Data = map[string][]byte{
			"new": []byte("data"),
		}
	}

	if labeled {
		secret.SetLabels(map[string]string{
			labelKey: labelValue,
		})
	}

	unstrObj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&secret)
	require.NoError(t, err)
	return &unstructured.Unstructured{Object: unstrObj}
}

func createFakeK8sCli(objects ...runtime.Object) (dynamic.Interface, func(t *testing.T, ctx context.Context)) {
	gvrs := sync.Map{}
	dynamicCli := fake.NewSimpleDynamicClient(scheme.Scheme, objects...)
	dynamicCli.PrependWatchReactor("*", func(action clienttesting.Action) (handled bool, ret watch.Interface, err error) {
		gvr := action.GetResource()
		ns := action.GetNamespace()
		watch, err := dynamicCli.Tracker().Watch(gvr, ns)
		if err != nil {
			return false, nil, err
		}

		gvrs.Store(gvr, struct{}{})
		return true, watch, nil
	})

	waitForGVRsWatchFn := func(t *testing.T, ctx context.Context) {
		// "(...) Any writes to the client after the informer's initial LIST and before the informer establishing the
		// watcher will be missed by the informer.
		// Source: https://github.com/kubernetes/client-go/blob/master/examples/fake-client/main_test.go#L74-L81
		//
		// So here, we wait for the informer to establish the watcher for both Secrets and ConfigMaps.
		err := wait.PollUntilContextCancel(ctx, 100*time.Millisecond, true, func(ctx context.Context) (done bool, err error) {
			gvrsLen := 0
			gvrs.Range(func(key, value any) bool {
				gvrsLen++
				return true
			})
			if gvrsLen == 2 {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err)
	}

	return dynamicCli, waitForGVRsWatchFn
}
