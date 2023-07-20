package logs

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/morikuni/aec"

	"github.com/kubeshop/botkube/internal/cli"
)

// Printer knows how to print Botkube logs.
type Printer struct {
	podName string
	newLog  chan string
	stop    chan struct{}
	parser  JSONParser
	logger  *log.Logger
}

// NewPrinter creates a new Printer instance.
func NewPrinter(podName string) *Printer {
	return &Printer{
		newLog: make(chan string, 10),
		stop:   make(chan struct{}),
		logger: log.NewWithOptions(os.Stdout, log.Options{
			Formatter: log.TextFormatter,
		}),
		podName: podName,
		parser:  JSONParser{},
	}
}

func (f *Printer) PrintLine(line string) {
	fields, lvl := f.parser.ParseLineIntoCharm(line)
	if fields == nil { // it was not recognized as JSON log entry, so let's print it as plain text.
		f.printLogLine(line)
		return
	}
	if lvl == log.DebugLevel && !cli.VerboseMode.IsEnabled() {
		return
	}

	fmt.Print(aec.EraseLine(aec.EraseModes.Tail))
	fmt.Print(aec.Column(6))
	f.logger.With(fields...).Print(nil)
}

func (f *Printer) printLogLine(line string) {
	fmt.Print(aec.EraseLine(aec.EraseModes.Tail))
	fmt.Print(aec.Column(6))
	fmt.Print(line)
}
