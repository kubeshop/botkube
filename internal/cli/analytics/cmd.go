package analytics

import "github.com/spf13/cobra"

const (
	// OptOutAnalyticsFlag is the name of the flag that can be used to opt out of analytics.
	OptOutAnalyticsFlag = "opt-out-analytics"

	// OptOutAnalyticsFlagUsage is the usage text for the OptOutAnalyticsFlag.
	OptOutAnalyticsFlagUsage = "The Botkube CLI tool collects anonymous usage analytics. This data is only available to the Botkube authors and helps us improve the tool."
)

// InjectAnalyticsReporting injects analytics reporting into the command.
func InjectAnalyticsReporting(in cobra.Command, cmdName string) *cobra.Command {
	reporter := NewReporter()

	in.PreRun = func(cmd *cobra.Command, args []string) {
		flags := in.Flags()
		skipAnalytics, err := flags.GetBool(OptOutAnalyticsFlag)
		if err != nil {
			return
		}
		if skipAnalytics {
			return
		}
		_ = reporter.ReportCommand(cmdName)
	}

	runner := in.RunE
	in.RunE = func(cmd *cobra.Command, args []string) error {
		err := runner(cmd, args)
		if err != nil {
			flags := in.Flags()
			skipAnalytics, flagErr := flags.GetBool(OptOutAnalyticsFlag)
			if flagErr != nil {
				return err
			}
			if !skipAnalytics {
				_ = reporter.ReportError(err, cmdName)
				// On error PostRun is not called, so we need to close the reporter here.
				reporter.Close()
			}
		}
		return err
	}
	in.PostRun = func(cmd *cobra.Command, args []string) {
		reporter.Close()
	}
	return &in
}
