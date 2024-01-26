package migrate

import (
	"fmt"

	"github.com/Masterminds/semver/v3"
	"go.szostok.io/version"
)

const (
	botkubeMinVersionConstraint = ">= 1.0"
)

// BotkubeVersionConstraints returns Botkube version constraints as a string.
func BotkubeVersionConstraints() string {
	botkubeMaxVersionConstraint := ""

	cliVer := version.Get().Version
	cliVersion, err := semver.NewVersion(cliVer)
	if err == nil {
		botkubeMaxVersionConstraint = fmt.Sprintf(", <= %s", cliVersion.String())
	}

	return fmt.Sprintf("%s%s", botkubeMinVersionConstraint, botkubeMaxVersionConstraint)
}

// IsCompatible checks if Botkube version is compatible with the migrate command.
func IsCompatible(botkubeVersionConstraintsStr string, botkubeVersionStr string) (bool, error) {
	constraint, err := semver.NewConstraint(botkubeVersionConstraintsStr)
	if err != nil {
		return false, fmt.Errorf("unable to parse Botkube semver version constraints: %w", err)
	}

	botkubeVersion, err := semver.NewVersion(botkubeVersionStr)
	if err != nil {
		return false, fmt.Errorf("unable to parse botkube version %s as semver: %w", botkubeVersion, err)
	}

	return constraint.Check(botkubeVersion), nil
}
