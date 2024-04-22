package execute

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/pflag"
)

const (
	cantParseCmd           = "cannot parse command. Please use 'help' to see supported commands"
	incorrectParamFlag     = "incorrect use of %s flag: %s"
	missingCmdParamValue   = `incorrect use of %s flag: an argument is missing. use %s="value" or %s value`
	multipleParams         = "incorrect use of %s flag: found more than one %s flag"
	paramFlagParseErrorMsg = `incorrect use of %s flag: could not parse flag in %s
error: %s
Use %s="value" or %s value`
)

// Flags contains cmd line arguments for executors.
type Flags struct {
	CleanCmd     string
	Filter       string
	ClusterName  string
	TokenizedCmd []string
	CmdHeader    string
}

// ParseFlags parses raw cmd and removes optional params with flags.
func ParseFlags(cmd string) (Flags, error) {
	cmd = ensureEvenSingleQuotes(cmd)

	cmd, clusterName, err := extractParam(cmd, "cluster-name")
	if err != nil {
		return Flags{}, err
	}
	cmd, filter, err := extractParam(cmd, "filter")
	if err != nil {
		return Flags{}, err
	}

	// all-clusters flag is the flag used when multiple clusters are connected to the same instance,
	// we need to remove it so other commands won't report problems with unknown flag.
	cmd, _, err = extractBoolParam(cmd, "all-clusters")
	if err != nil {
		return Flags{}, fmt.Errorf("while extracting all-cluster flag: %w", err)
	}

	cmd, cmdHeaderName, err := extractParam(cmd, "bk-cmd-header")
	if err != nil {
		return Flags{}, err
	}

	tokenized, err := shellwords.Parse(cmd)
	if err != nil {
		return Flags{}, err
	}
	return Flags{
		CleanCmd:     cmd,
		Filter:       filter,
		ClusterName:  clusterName,
		TokenizedCmd: tokenized,
		CmdHeader:    cmdHeaderName,
	}, nil
}

func extractParam(cmd, flagName string) (string, string, error) {
	flag := fmt.Sprintf("--%s", flagName)
	var withParam string
	var params []string
	args, _ := shellwords.Parse(cmd)
	f := pflag.NewFlagSet("extract-params", pflag.ContinueOnError)
	f.BoolP("help", "h", false, "to make sure that parsing is ignoring the --help,-h flags")

	f.ParseErrorsWhitelist.UnknownFlags = true
	f.StringArrayVar(&params, flagName, []string{}, "Output filter")
	if err := f.Parse(args); err != nil {
		return "", "", fmt.Errorf(incorrectParamFlag, flag, err)
	}

	if len(params) > 1 {
		return "", "", fmt.Errorf(multipleParams, flag, flagName)
	}

	if len(params) == 1 {
		withParam = params[0]
		if strings.HasPrefix(params[0], "-") {
			return "", "", fmt.Errorf(missingCmdParamValue, flag, flag, flag)
		}
	}

	for _, paramVal := range params {
		escapedParamVal := regexp.QuoteMeta(paramVal)
		paramFlagRegex, err := regexp.Compile(fmt.Sprintf(`%s[=|(' ')]*('%s'|"%s"|%s)("|')*`,
			flag,
			escapedParamVal,
			escapedParamVal,
			escapedParamVal))
		if err != nil {
			return "", "", fmt.Errorf("could not extract provided %s", flagName)
		}

		matches := paramFlagRegex.FindStringSubmatch(cmd)
		if len(matches) == 0 {
			return "", "", fmt.Errorf(paramFlagParseErrorMsg, flag, cmd, "it contains unsupported characters.", flag, flag)
		}
		cmd = strings.Replace(cmd, fmt.Sprintf(" %s", matches[0]), "", -1)
	}
	return cmd, withParam, nil
}

// ensureEvenSingleQuotes ensures that single quotes are even. It is required, e.g., for AI plugins to work.
// Otherwise, the command won't be parsed correctly (removing cluster name flags, etc.) and will result in an error.
// This is only a workaround for now. Ultimately, we should find a better and more generic way for extracting
// parameters. Additionally, we should delegate command tokenizing to the plugin.
func ensureEvenSingleQuotes(cmd string) string {
	for _, k := range []string{`‘`, `'`, `’`} {
		no := strings.Count(cmd, k)
		if no%2 == 0 {
			continue
		}
		cmd = strings.Replace(cmd, k, k+k, 1)
	}
	return cmd
}

func extractBoolParam(cmd, flagName string) (string, bool, error) {
	flag := fmt.Sprintf("--%s", flagName)
	var isSet bool
	args, _ := shellwords.Parse(cmd)
	f := pflag.NewFlagSet("extract-params", pflag.ContinueOnError)
	f.BoolP("help", "h", false, "to make sure that parsing is ignoring the --help,-h flags")

	f.ParseErrorsWhitelist.UnknownFlags = true
	f.BoolVar(&isSet, flagName, false, "Boolean flag")
	if err := f.Parse(args); err != nil {
		return "", false, fmt.Errorf(incorrectParamFlag, flag, err)
	}

	for _, val := range []string{"true", "false"} {
		paramFlagRegex, err := regexFlag(flag, val)
		if err != nil {
			return "", false, fmt.Errorf("could not extract provided %s", flagName)
		}

		matches := paramFlagRegex.FindStringSubmatch(cmd)
		if len(matches) == 0 {
			continue
		}
		cmd = strings.Replace(cmd, fmt.Sprintf(" %s", matches[0]), "", -1)
	}

	cmd = strings.ReplaceAll(cmd, flag, "")
	return cmd, isSet, nil
}

func regexFlag(flag, escapedParamVal string) (*regexp.Regexp, error) {
	return regexp.Compile(fmt.Sprintf(`%s[=|(' ')]*('%s'|"%s"|%s)("|')*`,
		flag,
		escapedParamVal,
		escapedParamVal,
		escapedParamVal))
}
