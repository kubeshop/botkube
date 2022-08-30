package meme

import (
	"fmt"
	"image"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/jpoz/gomeme"
)

// Generator provides a functionality to generate meme with random quote.
type Generator struct {
	quoteCli     *QuoteClient
	randomSource *rand.Rand
}

// NewGenerator returns a new Generator instance.
func NewGenerator(quoteCli *QuoteClient) *Generator {
	return &Generator{quoteCli: quoteCli,
		randomSource: rand.New(rand.NewSource(int64(time.Now().Nanosecond())))}
}

// Get returns a meme.
func (g *Generator) Get() (io.Reader, error) {
	quote, err := g.quoteCli.Get()
	if err != nil {
		return nil, fmt.Errorf("while getting quote for meme: %w", err)
	}

	imgPath := fmt.Sprintf("assets/face-%d.jpg", g.randomSource.Intn(7))
	file, err := os.Open(imgPath)
	if err != nil {
		return nil, fmt.Errorf("while opening image with path: %q: %w", imgPath, err)
	}
	defer file.Close()

	inputImage, _, err := image.Decode(file)
	if err != nil {
		return nil, fmt.Errorf("while decoding image %q: %w", imgPath, err)
	}

	config := gomeme.NewConfig()
	config.BottomText = quote
	meme := &gomeme.Meme{
		Config:   config,
		Memeable: gomeme.JPEG{Image: inputImage},
	}

	pReader, pWriter := io.Pipe()
	go func() {
		err := meme.Write(pWriter)
		pWriter.CloseWithError(err)
	}()
	return pReader, nil
}
