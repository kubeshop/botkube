package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-shellwords"

	"github.com/kubeshop/botkube/internal/loggerx"
)

func main() {
	pluginTargets := flag.String("plugin-targets", getEnv("PLUGIN_TARGETS", ""), "Comma separated list of specific targets to build. If empty, all targets are built.")
	outputMode := flag.String("output-mode", getEnv("OUTPUT_MODE", "binary"), "Output format. Allowed values: binary or archive.")
	buildSingle := flag.Bool("single-platform", os.Getenv("SINGLE_PLATFORM") != "", "If specified, builds only for current GOOS and GOARCH.")

	flag.Parse()

	switch *outputMode {
	case "archive":
		if *pluginTargets != "" {
			log.Fatal("Cannot build specific targets in archive mode")
		}
		// to produce archives, we need to run release instead of build
		releaseWithoutPublishing()
	case "binary":
		buildPlugins(*pluginTargets, *buildSingle)
	}
}

func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

func buildPlugins(pluginTargets string, single bool) {
	command := "goreleaser build -f .goreleaser.plugin.yaml --clean --snapshot"
	targets := strings.Split(pluginTargets, ",")
	for _, target := range targets {
		command += fmt.Sprintf(" --id %s", target)
	}

	if single {
		command += "--single-target"
	}

	runCommand(command)
}

func runCommand(command string) {
	args, err := shellwords.Parse(command)
	loggerx.ExitOnError(err, "while parsing command")

	bin, binArgs := args[0], args[1:]

	//nolint:gosec
	cmd := exec.Command(bin, binArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	loggerx.ExitOnError(cmd.Run(), "while running command")
}

func releaseWithoutPublishing() {
	runCommand("goreleaser release -f .goreleaser.plugin.yaml --clean --snapshot")
}
