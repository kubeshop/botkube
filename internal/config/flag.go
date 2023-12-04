package config

import (
	"github.com/spf13/pflag"
)

var configPathsFlag []string

// RegisterFlags registers config related flags.
func RegisterFlags(flags *pflag.FlagSet) {
	flags.StringSliceVarP(&configPathsFlag, "config", "c", nil, "Specify configuration file in YAML format (can specify multiple).")
}
