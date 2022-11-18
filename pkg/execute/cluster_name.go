package execute

import (
	"regexp"
	"strconv"
)

// getClusterNameFromKubectlCmd gets cluster name from kubectl command.
func getClusterNameFromKubectlCmd(cmd string) string {
	r, _ := regexp.Compile(`--cluster-name[=|' ']([^\s]*)`)
	//this gives 2 match with cluster name and without
	matchedArray := r.FindStringSubmatch(cmd)
	var s string
	if len(matchedArray) >= 2 {
		s = matchedArray[1]
	}

	str, err := strconv.Unquote(s)
	if err != nil {
		return s
	}

	return str
}
