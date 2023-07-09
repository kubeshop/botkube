package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// VerboseMode defines if CLI should use verbose mode
var VerboseMode = VerboseModeDisabled

// VerboseModeFlag is a type for capturing supported verbose mode formats.
// Implements pflag.Value interface.
type VerboseModeFlag int

const (
	// VerboseModeDisabled represents disabled verbose mode
	VerboseModeDisabled VerboseModeFlag = 0
	// VerboseModeSimple represents simple verbose mode (human friendly)
	VerboseModeSimple VerboseModeFlag = 1
	// VerboseModeTracing represents tracing verbose mode (output may be overwhelming)
	// In this mode http calls (request, response body, headers etc.) are logged
	VerboseModeTracing VerboseModeFlag = 2
)

// VerboseModeHumanMapping holds mapping between IDs and human-readable modes.
var VerboseModeHumanMapping = map[VerboseModeFlag]string{
	VerboseModeDisabled: "disable",
	VerboseModeSimple:   "simple",
	VerboseModeTracing:  "trace",
}

// ErrInvalidFormatType is returned when an unsupported verbose mode is used.
var ErrInvalidFormatType = fmt.Errorf("unknown verbose mode")

// RegisterVerboseModeFlag registers VerboseMode flag.
func RegisterVerboseModeFlag(flags *pflag.FlagSet) {
	flags.VarP(&VerboseMode, "verbose", "v", fmt.Sprintf("Prints more verbose output. Allowed values: %s", VerboseMode.AllowedOptions()))
	flags.Lookup("verbose").NoOptDefVal = VerboseModeHumanMapping[VerboseModeSimple]
}

// IsValid returns true if VerboseModeFlag is valid.
func (o VerboseModeFlag) IsValid() bool {
	switch o {
	case VerboseModeDisabled, VerboseModeSimple, VerboseModeTracing:
		return true
	}
	return false
}

// AllowedOptions returns list of allowed verbose mode options.
func (o VerboseModeFlag) AllowedOptions() string {
	return fmt.Sprintf("%s, %s, %s", VerboseModeDisabled, VerboseModeSimple, VerboseModeTracing)
}

// String returns the string representation of the Format. Required by pflag.Value interface.
func (o VerboseModeFlag) String() string {
	return fmt.Sprintf("%d - %s", o, VerboseModeHumanMapping[o])
}

// Set format type to a given input. Required by pflag.Value interface.
func (o *VerboseModeFlag) Set(in string) error {
	// try human
	for key, humanVal := range VerboseModeHumanMapping {
		if !strings.EqualFold(in, humanVal) {
			continue
		}
		*o = key
		return nil
	}
	// try int ID
	id, err := strconv.Atoi(in)
	if err != nil {
		return ErrInvalidFormatType
	}
	*o = VerboseModeFlag(id)
	if !o.IsValid() {
		return ErrInvalidFormatType
	}
	return nil
}

// Type returns data type. Required by pflag.Value interface.
func (o *VerboseModeFlag) Type() string {
	return "int/string"
}

// IsEnabled returns true if any verbose mode is enabled.
func (o VerboseModeFlag) IsEnabled() bool {
	return o != VerboseModeDisabled
}

// IsTracing returns true if tracing verbose mode is enabled.
func (o VerboseModeFlag) IsTracing() bool {
	return o == VerboseModeTracing
}
