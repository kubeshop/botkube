package analytics

import "github.com/spf13/cobra"

// InjectAnalyticsReporting injects analytics reporting into the command.
func InjectAnalyticsReporting(in cobra.Command, cmdName string) *cobra.Command {
	reporter := GetReporter()

	in.PreRun = func(cmd *cobra.Command, args []string) {
		// do not crash on telemetry errors
		_ = reporter.ReportCommand(cmdName)
		reporter.Close()
	}

	runner := in.RunE
	in.RunE = func(cmd *cobra.Command, args []string) error {
		err := runner(cmd, args)
		if err != nil {
			// do not crash on telemetry errors
			_ = reporter.ReportError(err, cmdName)
			reporter.Close()
		}
		return err
	}
	return &in
}
