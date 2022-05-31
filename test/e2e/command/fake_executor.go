package command

import (
	"fmt"
	"strings"
)

// FakeK8sServerVersion fake version send in ping response
const FakeK8sServerVersion = "v1.15.3"

// FakeKubectlResponse map for fake Kubectl responses
var FakeKubectlResponse = map[string]string{
	"/usr/local/bin/kubectl -n default get pods": "NAME                           READY   STATUS    RESTARTS   AGE\n" +
		"nginx-xxxxxxx-yyyyyyy          1/1     Running   1          1d",
	"sh -c /usr/local/bin/kubectl version --short=true | grep Server": fmt.Sprintf("Server Version: %s\n", FakeK8sServerVersion),
}

// FakeCommandRunnerFunc mocks real execute.CommandRunnerFunc
func FakeCommandRunnerFunc(command string, args []string) (string, error) {
	cmd := fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	return FakeKubectlResponse[cmd], nil
}
