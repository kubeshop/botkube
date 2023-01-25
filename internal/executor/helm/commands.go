package helm

// Commands defines all supported Helm plugin commands and their flags.
type Commands struct {
	Install  *InstallCommand  `arg:"subcommand:install"`
	Version  *VersionCommand  `arg:"subcommand:version"`
	Status   *StatusCommand   `arg:"subcommand:status"`
	Test     *TestCommand     `arg:"subcommand:test"`
	Rollback *RollbackCommand `arg:"subcommand:rollback"`
	Upgrade  *UpgradeCommand  `arg:"subcommand:upgrade"`
	Help     *HelpCommand     `arg:"subcommand:help"`
	Get      *GetCommand      `arg:"subcommand:get"`

	// embed on the root of the Command struct to inline all aliases.
	HistoryCommandAliases
	UninstallCommandAliases
	ListCommandAliases

	GlobalFlags
}

// GlobalFlags holds flags supported by all Helm plugin commands
type GlobalFlags struct {
	Namespace  string `arg:"--namespace,-n"`
	Debug      bool   `arg:"--debug"`
	BurstLimit int    `arg:"--burst-limit"`
}

type noopValidator struct{}

// Validate does nothing. It can be used if no validation is required,
// but you want to satisfy the command interface.
func (noopValidator) Validate() error {
	return nil
}
