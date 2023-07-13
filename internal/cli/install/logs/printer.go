package logs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/log"
	charmlog "github.com/charmbracelet/log"
	"github.com/morikuni/aec"
	"github.com/muesli/termenv"

	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/cli/printer"
)

type FixedHeightPrinter struct {
	height      int
	logsBuffer  []string
	podPhase    string
	podName     string
	newLog      chan string
	newPodPhase chan string
	stop        chan struct{}
	parser      JSONParser
	logger      *log.Logger
}

func NewFixedHeightPrinter(height int, name string) *FixedHeightPrinter {
	return &FixedHeightPrinter{
		height:      height,
		logsBuffer:  []string{},
		newLog:      make(chan string, 10),
		newPodPhase: make(chan string, 10),
		stop:        make(chan struct{}),
		logger: log.NewWithOptions(os.Stdout, log.Options{
			Formatter: log.TextFormatter,
		}),
		podName: name,
		parser:  JSONParser{},
	}
}

func (f *FixedHeightPrinter) Start(ctx context.Context) {
	refreshDuration := 100 * time.Millisecond
	idleDelay := time.NewTimer(refreshDuration)
	defer idleDelay.Stop()

	buff := bytes.Buffer{}
	status := printer.NewStatus(&buff, "")
	status.Step("Streaming logs...")
	termenv.SaveCursorPosition()
	for {
		f.printData(buff.String() + "\n") // it's without new line when it's in progress
		idleDelay.Reset(refreshDuration)

		select {
		case <-f.stop:
			status.End(true)
			termenv.RestoreCursorPosition()
			f.printStatusHeader(buff.String())
			f.printLogs(true)
			fmt.Println()
			return
		case <-ctx.Done():
			status.End(false)
			termenv.RestoreCursorPosition()

			f.printStatusHeader(buff.String())
			f.printLogs(true)
			fmt.Println()
			return
		case <-idleDelay.C:
		case entry := <-f.newLog:
			f.logsBuffer = append(f.logsBuffer, entry)
			if len(f.logsBuffer) > f.height {
				//now we need to simulate scrolling, so all lines are moved N-1, where the first line is just removed.
				f.logsBuffer = f.logsBuffer[1:]
			}
		case podPhase := <-f.newPodPhase:
			f.podPhase = podPhase
		}

		fmt.Print(aec.Up(uint(f.height + 4)))
	}
}

func (f *FixedHeightPrinter) printData(header string) {
	f.printStatusHeader(header)
	f.printLogs(false)
}
func (f *FixedHeightPrinter) printStatusHeader(step string) {
	fmt.Println(step)
	fmt.Printf("    Pods: %s Phase: %s\n", f.podName, f.podPhase)
	fmt.Println()
}

func (f *FixedHeightPrinter) UpdatePodPhase(phase string) {
	select {
	case f.newPodPhase <- phase:
	default:
	}
}

func (f *FixedHeightPrinter) printLogs(skip bool) {
	wroteLines := 0
	for _, item := range f.logsBuffer {
		fields, lvl := f.parser.ParseLineIntoCharm(item)
		if fields == nil {
			wroteLines++
			f.printLogLine(item)
			continue
		}
		if lvl == charmlog.DebugLevel && !cli.VerboseMode.IsEnabled() {
			continue
		}
		wroteLines++
		fmt.Print(aec.EraseLine(aec.EraseModes.Tail))
		fmt.Print(aec.Column(6))
		f.logger.With(fields...).Print(nil)
	}

	if skip {
		return
	}
	for i := wroteLines; i < f.height; i++ {
		f.printLogLine("\n")
	}
}

func (f *FixedHeightPrinter) printLogLine(line string) {
	fmt.Print(aec.EraseLine(aec.EraseModes.Tail))
	fmt.Print(aec.Column(6))
	fmt.Print(line)
}

func (f *FixedHeightPrinter) AppendLogEntry(entry string) {
	if strings.TrimSpace(entry) == "" {
		return
	}
	select {
	case f.newLog <- entry:
	default:
	}
}

func (f *FixedHeightPrinter) Stop() {
	close(f.stop)
}
