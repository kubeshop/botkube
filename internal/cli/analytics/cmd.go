package analytics

import "github.com/spf13/cobra"

// InjectAnalyticsReporting injects analytics reporting into the command.
func InjectAnalyticsReporting(in cobra.Command, cmdName string) *cobra.Command {
	runner := in.RunE
	in.RunE = func(cmd *cobra.Command, args []string) error {
		reporter := GetReporter(in)
		defer reporter.Close()

		// do not crash on telemetry errors
		_ = reporter.ReportCommand(cmdName)

		err := runner(cmd, args)
		if err != nil {
			// do not crash on telemetry errors
			_ = reporter.ReportError(err, cmdName)

		}
		return err
	}
	return &in
}
