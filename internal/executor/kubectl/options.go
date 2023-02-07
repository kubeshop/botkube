package kubectl

import "github.com/MakeNowJust/heredoc"

func optionsCommandOutput() string {
	return heredoc.Doc(`
			The following options can be passed to any command:

			--match-server-version=false:
				Require server version to match client version

			-n, --namespace='':
				If present, the namespace scope for this CLI request

			--request-timeout='0':
				The length of time to wait before giving up on a single server request. Non-zero values should contain a
				corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests.

			-v, --v=0:
				number for the log level verbosity

			--warnings-as-errors=false:
				Treat warnings received from the server as errors and exit with a non-zero exit code
			`)
}
