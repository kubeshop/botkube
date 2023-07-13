package logs

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/avast/retry-go/v4"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	containerName = "botkube"
)

// StartsLogsStreaming reads the data from request and writes into
// the out channel. It buffers data from requests until the newline or io.EOF
// occurs in the data, so it doesn't interleave logs sub-line
// when running concurrently.
//
// A successful read returns err == nil, not err == io.EOF.
// Because the function is defined to read from request until io.EOF, it does
// not treat an io.EOF as an error to be reported.
func StartsLogsStreaming(ctx context.Context, clientset *kubernetes.Clientset, namespace, name string, out chan<- []byte) error {
	return retry.Do(func() error {
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

		r := bufio.NewReader(podLogs)
		for {
			bytes, readErr := r.ReadBytes('\n')
			// write first, as there might be some chars already loaded
			out <- bytes
			//if strings.Contains(string(bytes), "Botkube connected to") {
			//	return nil
			//}
			switch readErr {
			case nil:
			case io.EOF:
				return nil
			case ctx.Err():
				return nil
			default:
				return fmt.Errorf("while reading log stream: %v", readErr)
			}
		}
	}, retry.Attempts(3), retry.Delay(time.Second))
}
