package helm

import (
	"fmt"
	"reflect"
	"strings"
)

const tagArgName = "arg"

func renderSupportedFlags(in any) string {
	var flags []string
	fields := reflect.VisibleFields(reflect.TypeOf(in))
	for _, field := range fields {
		flagName, _ := field.Tag.Lookup(tagArgName)
		flags = append(flags, flagName)
	}

	return strings.Join(flags, "\n")
}

func returnErrorOfAllSetFlags(in any) error {
	var setFlags []string
	vv := reflect.ValueOf(in)
	fields := reflect.VisibleFields(reflect.TypeOf(in))

	for _, field := range fields {
		flagName, _ := field.Tag.Lookup(tagArgName)
		if vv.FieldByIndex(field.Index).IsZero() {
			continue
		}

		setFlags = append(setFlags, flagName)
	}

	if len(setFlags) > 0 {
		return newUnsupportedFlagsError(setFlags)
	}

	return nil
}

func newUnsupportedFlagsError(flags []string) error {
	if len(flags) == 1 {
		return fmt.Errorf("The %q flag is not supported by the Botkube Helm plugin. Please remove it.", flags[0])
	}

	points := make([]string, len(flags))
	for i, err := range flags {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Errorf(
		"Those flags are not supported by the Botkube Helm Plugin:\n\t%s\nPlease remove them.",
		strings.Join(points, "\n\t"))
}
