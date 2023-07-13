package logs

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/log"
	"github.com/morikuni/aec"

	"github.com/kubeshop/botkube/internal/cli/printer"
)

const extendedKitchenTimeFormat = "3:04:05 PM"

type FixedHeightPrinter struct {
	height         int
	logsBuffer     []string
	moveUpByHeight aec.ANSI
	parser         *KVParser
	logger         *log.Logger
	podPhase       string
	sync.Mutex
	alreadyUsed bool
	podName     string
	newLog      chan string
	newPodPhase chan string
	stop        chan struct{}
}

func NewFixedHeightPrinter(height int, parser *KVParser, name string) *FixedHeightPrinter {
	return &FixedHeightPrinter{
		height:         height,
		logsBuffer:     []string{},
		moveUpByHeight: aec.Up(uint(height)),
		parser:         parser,
		alreadyUsed:    false,
		newLog:         make(chan string, 10),
		newPodPhase:    make(chan string, 10),
		logger: log.NewWithOptions(os.Stdout, log.Options{
			TimeFormat: extendedKitchenTimeFormat,
			Formatter:  log.TextFormatter,
		}),
		stop:    make(chan struct{}),
		podName: name,
	}
}

func (f *FixedHeightPrinter) Start(ctx context.Context) {
	refreshDuration := 100 * time.Millisecond
	idleDelay := time.NewTimer(refreshDuration)
	defer idleDelay.Stop()

	buff := bytes.Buffer{}
	status := printer.NewStatus(&buff, "")
	status.Step("Streaming logs...")
	f.printStatusHeader(buff.String() + "\n") // it's without new line when it's in progress

	resetHeader := func() {
		fmt.Print(aec.Save)
		fmt.Print(aec.Up(uint(len(f.logsBuffer) + 4)))
		f.printStatusHeader(buff.String() + "\n") // it's without new line when it's in progress
		fmt.Print(aec.Restore)
	}

	for {
		idleDelay.Reset(refreshDuration)

		select {
		case <-f.stop:
			status.End(true)
			fmt.Print(aec.Up(uint(len(f.logsBuffer) + 4)))
			f.printStatusHeader(buff.String())
			f.printLogs()
			fmt.Println()
			return
		case <-ctx.Done():
			status.End(false)
			fmt.Print(aec.Up(uint(len(f.logsBuffer) + 4)))
			f.printStatusHeader(buff.String())
			f.printLogs()
			fmt.Println()
			return
		case <-idleDelay.C:
			resetHeader()
		case entry := <-f.newLog:
			f.logsBuffer = append(f.logsBuffer, entry)
			if len(f.logsBuffer) <= f.height {
				f.printLogLine(entry)
				continue
			}

			// now we need to simulate scrolling, so all lines are moved N-1, where the first line is just removed.
			f.logsBuffer = f.logsBuffer[1:]
			fmt.Print(aec.Up(uint(len(f.logsBuffer))))
			f.printLogs()
		case podPhase := <-f.newPodPhase:
			f.podPhase = podPhase
			resetHeader()
		}
	}
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

func (f *FixedHeightPrinter) printLogs() {
	for _, item := range f.logsBuffer {
		f.printLogLine(item)
	}
}

func (f *FixedHeightPrinter) printLogLine(line string) {
	fmt.Print(aec.Column(6))
	fmt.Print(aec.EraseLine(aec.EraseModes.Tail))
	//fmt.Print(item)
	fields := f.parser.ParseLineIntoCharm(line)
	f.logger.With(fields...).Print(nil)
}

func (f *FixedHeightPrinter) AppendLogEntry(entry string) {
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
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
