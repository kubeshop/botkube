package version

// Version The below variables are overridden using the build process
// name of the release
var Version = "dev"

// GitCommitID git commit id of the release
var GitCommitID = "none"

// BuildDate date for the release
var BuildDate = "unknown"

// Short returns short version of the release
func Short() string {
	return Version
}

// Details struct contains data about a given version.
type Details struct {
	Version     string `json:"version"`
	GitCommitID string `json:"gitCommit"`
	BuildDate   string `json:"buildDate"`
}

// Info returns Details struct with version info.
func Info() Details {
	return Details{
		Version:     Version,
		GitCommitID: GitCommitID,
		BuildDate:   BuildDate,
	}
}
