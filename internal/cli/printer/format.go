package printer

import (
	"fmt"
)

// PrintFormat is a type for capturing supported output formats.
// Implements pflag.Value interface.
type PrintFormat string

// ErrInvalidFormatType is returned when an unsupported format type is used
var ErrInvalidFormatType = fmt.Errorf("invalid output format type")

const (
	// JSONFormat represents JSON data format.
	JSONFormat PrintFormat = "json"
	// YAMLFormat represents YAML data format.
	YAMLFormat PrintFormat = "yaml"
)

// IsValid returns true if PrintFormat is valid.
func (o PrintFormat) IsValid() bool {
	switch o {
	case JSONFormat, YAMLFormat:
		return true
	}
	return false
}

// String returns the string representation of the Format. Required by pflag.Value interface.
func (o PrintFormat) String() string {
	return string(o)
}

// Set format type to a given input. Required by pflag.Value interface.
func (o *PrintFormat) Set(in string) error {
	*o = PrintFormat(in)
	if !o.IsValid() {
		return ErrInvalidFormatType
	}
	return nil
}

// Type returns data type. Required by pflag.Value interface.
func (o *PrintFormat) Type() string {
	return "string"
}
