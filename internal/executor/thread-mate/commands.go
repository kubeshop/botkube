package thread_mate

type (
	// Commands represents a collection of subcommands.
	Commands struct {
		Pick     *PickCmd     `arg:"subcommand:pick"`
		Get      *GetCmd      `arg:"subcommand:get"`
		Resolve  *ResolveCmd  `arg:"subcommand:resolve"`
		Takeover *TakeoverCmd `arg:"subcommand:takeover"`
	}

	// ResolveCmd represents the "resolve" subcommand.
	ResolveCmd struct {
		ID string `arg:"--id"`
	}

	// TakeoverCmd represents the "takeover" subcommand.
	TakeoverCmd struct {
		ID string `arg:"--id"`
	}

	// PickCmd represents the "pick" subcommand.
	PickCmd struct {
		MessageContext string `arg:"-m,--message"`
	}

	// GetCmd represents the "get" subcommand.
	GetCmd struct {
		Activity *ActivityCmd `arg:"subcommand:activity"`
	}
	
	// ActivityCmd represents the "activity" subcommand under the "get" command.
	ActivityCmd struct {
		AssigneeIDs string `arg:"--assignee-ids"`
		Type        string `arg:"--thread-type"`
	}
)
