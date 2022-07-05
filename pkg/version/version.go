package version

import "fmt"

// Version The below variables are overridden using the build process
// name of the release
var Version = "dev"

// GitCommitID git commit id of the release
var GitCommitID = "none"

// BuildDate date for the release
var BuildDate = "unknown"

const versionLongFmt = `{"Version": "%s", "GitCommit": "%s", "BuildDate": "%s"}`

// Long long version of the release
func Long() string {
	return fmt.Sprintf(versionLongFmt, Version, GitCommitID, BuildDate)
}

// Short short version of the release
func Short() string {
	return Version
}
