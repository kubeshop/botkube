package plugin

import (
	"context"
	"fmt"

	"github.com/MakeNowJust/heredoc"
)

func Example_parseCommand() {
	type (
		CommitCmd struct {
			All     bool   `arg:"-a"`
			Message string `arg:"-m"`
		}
		Commands struct {
			Commit *CommitCmd `arg:"subcommand:commit"`
			Quiet  bool       `arg:"-q"` // this flag is global to all subcommands
		}
	)

	rawCommand := "./example commit -m 'hakuna matata' -a -q"

	var cmd Commands
	err := ParseCommand("./example", rawCommand, &cmd)
	if err != nil {
		panic(err)
	}

	switch {
	case cmd.Commit != nil:
		fmt.Println(heredoc.Docf(`
			global quiet flag: %v
			commit message: %v
			commit all flag: %v
		`, cmd.Commit.All, cmd.Commit.Message, cmd.Quiet))
	}

	// output:
	// global quiet flag: true
	// commit message: hakuna matata
	// commit all flag: true
}

func Example_executeCommand() {
	out, err := ExecuteCommand(context.Background(), `echo "hakuna matata"`)
	if err != nil {
		panic(err)
	}

	fmt.Println(out.Stdout)

	// output:
	// hakuna matata
}

func Example_executeCommandWithEnv() {
	out, err := ExecuteCommand(context.Background(), `sh -c "echo ${CUSTOM_ENV}"`, ExecuteCommandEnvs(map[string]string{
		"CUSTOM_ENV": "magic-value",
	}))
	if err != nil {
		panic(err)
	}

	fmt.Println(out.Stdout)

	// output:
	// magic-value
}
