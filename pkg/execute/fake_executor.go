// +build test

package execute

import (
	"fmt"
	"strings"
)

// K8sVersion fake version send in ping response
const K8sVersion = "v1.15.3"

// KubectlResponse map for fake Kubectl responses
var KubectlResponse = map[string]string{
	"-n default get pods": "NAME                           READY   STATUS    RESTARTS   AGE\n" +
		"nginx-xxxxxxx-yyyyyyy          1/1     Running   1          1d",
	"-c " + kubectlBinary + " version --short=true | grep Server": fmt.Sprintf("Server Version: %s\n", K8sVersion),
}

// FakeRunner mocks Run
type FakeRunner struct {
	command string
	args    []string
}

// NewCommandRunner returns new DefaultRunner
func NewCommandRunner(command string, args []string) CommandRunner {
	return FakeRunner{
		command: command,
		args:    args,
	}
}

// Run executes bash command
func (r FakeRunner) Run() (string, error) {
	cmd := strings.Join(r.args, " ")
	return KubectlResponse[cmd], nil
}
