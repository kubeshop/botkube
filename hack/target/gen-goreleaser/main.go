package main

import (
	"fmt"
	"github.com/kubeshop/botkube/pkg/loggerx"
	"os"
	"text/template"
)

const (
	templateFile = "./.goreleaser.plugin.tpl.yaml"
	outputFile   = "./.goreleaser.plugin.yaml"
	entrypoint   = "./cmd"
	filePerm     = 0o644

	fileNoEditHeader = "# The code has been automatically generated and should not be modified directly. To update, run 'make gen-plugins-goreleaser' from the root directory of this repository."
)

type (
	Plugins []Plugin
	Plugin  struct {
		Name string
		Type string
	}
)

func main() {
	executors, err := os.ReadDir(entrypoint + "/executor")
	loggerx.ExitOnError(err, "collecting executors")
	sources, err := os.ReadDir(entrypoint + "/source")
	loggerx.ExitOnError(err, "collecting sources")

	var plugins Plugins
	for _, d := range executors {
		plugins = append(plugins, Plugin{
			Type: "executor",
			Name: d.Name(),
		})
	}
	for _, d := range sources {
		plugins = append(plugins, Plugin{
			Type: "source",
			Name: d.Name(),
		})
	}

	file, err := os.ReadFile(templateFile)
	loggerx.ExitOnError(err, "reading tpl file")

	//  Change delims to not interfere with the GoReleaser templates.
	tpl, err := template.New("goreleaser").Delims("<", ">").Parse(string(file))
	loggerx.ExitOnError(err, "creating tpl processor")

	dst, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, filePerm)
	loggerx.ExitOnError(err, "open destination file file")

	fmt.Fprintln(dst, fileNoEditHeader)
	fmt.Fprintln(dst)

	err = tpl.Execute(dst, plugins)
	loggerx.ExitOnError(err, "while running tpl processor")
}
