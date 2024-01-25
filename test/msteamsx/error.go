package msteamsx

import (
	"strings"
)

// IsMissingPermissionsError checks if error is related to missing permissions.
// See: https://github.com/microsoftgraph/msgraph-sdk-go/issues/510
// I tried to use underlying ApiError, but it's empty...
func IsMissingPermissionsError(err error) bool {
	return strings.Contains(err.Error(), "Missing role permissions on the request.")
}
