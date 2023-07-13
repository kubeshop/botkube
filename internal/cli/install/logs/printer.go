package logs

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/log"
	charmlog "github.com/charmbracelet/log"
	"github.com/morikuni/aec"
	"github.com/muesli/reflow/indent"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/printer"
)

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

// Start starts the log streaming process.
func (f *Printer) Start(ctx context.Context, status *printer.StatusPrinter) {
	status.InfoWithBody("Streaming logs...", indent.String(fmt.Sprintf("Pod: %s", f.podName), 4))
	fmt.Println()

	for {
		select {
		case <-f.stop:
			return
		case <-ctx.Done():
			status.Infof("Requested logs streaming cancel...")
			return
		case entry := <-f.newLog:
			f.printLogs(entry)
		}
	}
}

// AppendLogEntry appends a log entry to the printer.
func (f *Printer) AppendLogEntry(entry string) {
	if strings.TrimSpace(entry) == "" {
		return
	}
	select {
	case f.newLog <- entry:
	default:
	}
}

// Stop stops the printer.
func (f *Printer) Stop() {
	close(f.stop)
}

func (f *Printer) printLogs(item string) {
	fields, lvl := f.parser.ParseLineIntoCharm(item)
	if fields == nil {
		f.printLogLine(item)
		return
	}
	if lvl == charmlog.DebugLevel && !cli.VerboseMode.IsEnabled() {
		return
	}
	fmt.Print(aec.EraseLine(aec.EraseModes.Tail))
	fmt.Print(aec.Column(6))
	f.logger.With(fields...).Print(nil)
}

func (f *Printer) printLogLine(line string) {
	fmt.Print(aec.Column(6))
	fmt.Print(line)
}
