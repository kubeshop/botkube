package logs

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/morikuni/aec"
)

const extendedKitchenTimeFormat = "3:04:05 PM"

type FixedHeightPrinter struct {
	Height         int
	logsBuffer     []string
	moveUpByHeight aec.ANSI
	parser         *KVParser
	logger         *log.Logger
}

func NewFixedHeightPrinter(height int, parser *KVParser) *FixedHeightPrinter {
	return &FixedHeightPrinter{
		Height:         height,
		logsBuffer:     []string{},
		moveUpByHeight: aec.Up(uint(height)),
		parser:         parser,
		logger: log.NewWithOptions(os.Stdout, log.Options{
			TimeFormat: extendedKitchenTimeFormat,
			Formatter:  log.TextFormatter,
		}),
	}
}

func (f *FixedHeightPrinter) Print(entry string) {
	f.logsBuffer = append(f.logsBuffer, entry)

	if len(f.logsBuffer) <= f.Height { // just print a new line as we didn't reach max limit
		fmt.Print(aec.Column(5))
		fields := f.parser.ParseLineIntoCharm(f.logsBuffer[len(f.logsBuffer)-1])
		f.logger.With(fields...).Print(nil)
		return
	}

	// now we need to simulate scrolling, so all lines are moved N-1, where the first line is just removed.
	f.logsBuffer = f.logsBuffer[1:]
	fmt.Print(f.moveUpByHeight)
	for _, item := range f.logsBuffer {
		fmt.Print(aec.Column(5))
		fields := f.parser.ParseLineIntoCharm(item)
		f.logger.With(fields...).Print(nil)
	}
}
