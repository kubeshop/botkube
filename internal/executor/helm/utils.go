package helm

import (
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
