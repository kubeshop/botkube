package command

// Origin defines the origin of the command.
type Origin string

const (
	// UnknownOrigin is the default value for Origin.
	UnknownOrigin Origin = "unknown"

	// TypedOrigin is the value for Origin when the command was typed by the user.
	TypedOrigin Origin = "typed"

	// ButtonClickOrigin is the value for Origin when the command was triggered by a button click.
	ButtonClickOrigin Origin = "buttonClick"

	// SelectValueChangeOrigin is the value for Origin when the command was triggered by a select value change.
	SelectValueChangeOrigin Origin = "selectValueChange"

	// MultiSelectValueChangeOrigin is the value for Origin when the command was triggered by a multi-select value change.
	MultiSelectValueChangeOrigin Origin = "multiSelectValueChange"

	// PlainTextInputOrigin is the value for Origin when the command was triggered by a plain text input.
	PlainTextInputOrigin Origin = "plainTextInput"
)
