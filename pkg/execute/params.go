package execute

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/spf13/pflag"
)

const (
	cantParseCmd            = "cannot parse command. Please use 'help' to see supported commands"
	incorrectFilterFlag     = "incorrect use of --filter flag: %s"
	missingCmdFilterValue   = `incorrect use of --filter flag: an argument is missing. use --filter="value" or --filter value`
	multipleFilters         = "incorrect use of --filter flag: found more than one filter flag"
	filterFlagParseErrorMsg = `incorrect use of --filter flag: could not parse flag in %s
error: %s
Use --filter="value" or --filter value`
)

var (
	clusterNameFlagRegex = regexp.MustCompile(`--cluster-name[=|\s]*(\S*)`)
)

// Flags contains cmd line arguments for executors.
type Flags struct {
	CleanCmd     string
	Filter       string
	ClusterName  string
	TokenizedCmd []string
}

// ParseFlags parses raw cmd and removes optional params with flags.
func ParseFlags(cmd string) (Flags, error) {
	groups := clusterNameFlagRegex.FindAllStringSubmatch(cmd, -1)
	cmd, clusterName := extractParam(cmd, groups)

	cmd, filter, err := extractFilterParam(cmd)
	if err != nil {
		return Flags{}, err
	}
	tokenized, err := shellwords.Parse(strings.TrimSpace(cmd))
	if err != nil {
		return Flags{}, errors.New(cantParseCmd)
	}
	return Flags{
		CleanCmd:     cmd,
		Filter:       filter,
		ClusterName:  clusterName,
		TokenizedCmd: tokenized,
	}, nil
}

func extractParam(cmd string, groups [][]string) (string, string) {
	var param string
	if len(groups) > 0 && len(groups[0]) > 1 {
		param = groups[0][1]
		// remove quotation marks, if present
		if p, err := strconv.Unquote(groups[0][1]); err == nil {
			param = p
		}
	}
	for _, matches := range groups {
		for _, match := range matches {
			if match != "" {
				cmd = strings.Replace(cmd, fmt.Sprintf(" %s", match), "", 1)
			}
		}
	}
	return cmd, param
}

func extractFilterParam(cmd string) (string, string, error) {
	var withFilter string
	var filters []string
	args, _ := shellwords.Parse(cmd)
	f := pflag.NewFlagSet("extract-filters", pflag.ContinueOnError)
	f.BoolP("help", "h", false, "to make sure that parsing is ignoring the --help,-h flags")

	f.ParseErrorsWhitelist.UnknownFlags = true
	f.StringArrayVar(&filters, "filter", []string{}, "Output filter")
	if err := f.Parse(args); err != nil {
		return "", "", fmt.Errorf(incorrectFilterFlag, err)
	}

	if len(filters) > 1 {
		return "", "", errors.New(multipleFilters)
	}

	if len(filters) == 1 {
		withFilter = filters[0]
		if strings.HasPrefix(filters[0], "-") {
			return "", "", errors.New(missingCmdFilterValue)
		}
	}

	for _, filterVal := range filters {
		escapedFilterVal := regexp.QuoteMeta(filterVal)
		filterFlagRegex, err := regexp.Compile(fmt.Sprintf(`--filter[=|(' ')]*('%s'|"%s"|%s)("|')*`,
			escapedFilterVal,
			escapedFilterVal,
			escapedFilterVal))
		if err != nil {
			return "", "", errors.New("could not extract provided filter")
		}

		matches := filterFlagRegex.FindStringSubmatch(cmd)
		if len(matches) == 0 {
			return "", "", fmt.Errorf(filterFlagParseErrorMsg, cmd, "it contains unsupported characters.")
		}
		cmd = strings.Replace(cmd, fmt.Sprintf(" %s", matches[0]), "", -1)
	}
	return cmd, withFilter, nil
}
