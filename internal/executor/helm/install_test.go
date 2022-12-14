package helm

import (
	"fmt"
	"os"
	"testing"

	"github.com/alexflint/go-arg"
	"github.com/mattn/go-shellwords"
	"github.com/stretchr/testify/require"
)

type helm struct {
	Install *InstallCmd `arg:"subcommand:install"`
	Quiet   bool        `arg:"-q"` // this flag is global to all subcommands
}

func Test(t *testing.T) {

	var args helm
	p, err := arg.NewParser(arg.Config{
		Program: "helm",
	}, &args)
	require.NoError(t, err)

	argsss, _ := shellwords.Parse("install mynginx https://example.com/charts/nginx-1.2.3.tgz --create-namespace --dupa")
	err = p.Parse(argsss)
	switch err {
	case nil, arg.ErrHelp, arg.ErrVersion:
		// ignore
		fmt.Println("err", err)
	default:
		fmt.Println(p.SubcommandNames())
		require.NoError(t, err)
	}

	switch {
	case args.Install != nil:
		if err == arg.ErrHelp {
			fmt.Println("help")
			p.WriteHelp(os.Stdout)
		}
		err := args.Install.Validate()
		require.NoError(t, err)
		fmt.Println(args.Install.Atomic)
		//fmt.Println(args.Install.Chart)
		//fmt.Println(args.Install.CreateNamespace)
		//fmt.Println(args.Quiet)
	}

}
