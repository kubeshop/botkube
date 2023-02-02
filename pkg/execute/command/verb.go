package command

// Verb are commands supported by the bot.
type Verb string

const (
	PingVerb     Verb = "ping"
	HelpVerb     Verb = "help"
	VersionVerb  Verb = "version"
	FeedbackVerb Verb = "feedback"
	ListVerb     Verb = "list"
	EnableVerb   Verb = "enable"
	DisableVerb  Verb = "disable"
	EditVerb     Verb = "edit"
	StatusVerb   Verb = "status"
	ShowVerb     Verb = "show"
)

func AllVerbs() []Verb {
	return []Verb{
		PingVerb,
		HelpVerb,
		VersionVerb,
		FeedbackVerb,
		ListVerb,
		EnableVerb,
		DisableVerb,
		EditVerb,
		StatusVerb,
		ShowVerb,
	}
}
