package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	containerName = "botkube"
)

// DefaultConsumeRequest reads the data from request and writes into
// the out writer. It buffers data from requests until the newline or io.EOF
// occurs in the data, so it doesn't interleave logs sub-line
// when running concurrently.
//
// A successful read returns err == nil, not err == io.EOF.
// Because the function is defined to read from request until io.EOF, it does
// not treat an io.EOF as an error to be reported.
func DefaultConsumeRequest(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, out chan<- []byte, close <-chan struct{}) error {
	err := retry.Do(func() error {
		selector := labels.SelectorFromSet(map[string]string{"app": name}).String()
		list, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			TypeMeta:      metav1.TypeMeta{},
			LabelSelector: selector,
			Limit:         1,
		})
		if err != nil {
			return err
		}

		if len(list.Items) == 0 {
			return fmt.Errorf("there are no Pods for selector %q in namesapce %q", selector, namespace)
		}
		name = list.Items[0].Name
		return nil
	}, retry.Attempts(3), retry.Delay(time.Second))
	if err != nil {
		return err
	}

	err = retry.Do(func() error {
		req := clientset.CoreV1().Pods(namespace).GetLogs(name, &v1.PodLogOptions{
			Container:  containerName,
			Follow:     true,
			Timestamps: false,
		})
		podLogs, err := req.Stream(ctx)
		if err != nil {
			return fmt.Errorf("while opening log stream: %v", err)
		}
		defer podLogs.Close()

		//go func() {
		//	<-close
		//	podLogs.Close()
		//}()

		r := bufio.NewReader(podLogs)
		for {
			bytes, readErr := r.ReadBytes('\n')
			// write first, as there might be some chars already loaded
			out <- bytes
			if strings.Contains(string(bytes), "Botkube connected to") {
				return nil
			}
			switch readErr {
			case nil:
			case io.EOF:
				return nil
			default:
				return readErr
			}
		}
	}, retry.Attempts(3), retry.Delay(time.Second))
	if err != nil {
		return err
	}
	return nil
}
