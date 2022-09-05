package quote

import (
	"math/rand"
	"time"
)

var defaultQuotes = []string{
	"don't put your hand in boiling water",
	"do not breathe under the water",
	"breathing will help you live",
	"don't look up and spit",
	"in case of fire, exit building before tweeting about it",
	"don't take advice from posters",
	"don't eat yellow snow",
	"don't swim in waters inhabited by large alligators",
	"a day without sunshine is like, you know, night",
	"there isn't really a Nigerian Prince who wants to transfer money to you",
	"never buy a car you can’t push",
	"only ninja can sneak upon other ninja",
}

// Generator provides a functionality to generate a random quote.
type Generator struct {
	quotes       []string
	randomSource *rand.Rand
}

// NewGenerator returns a new Generator instance.
func NewGenerator(quotes []string) *Generator {
	if len(quotes) == 0 {
		quotes = defaultQuotes
	}
	return &Generator{
		quotes:       quotes,
		randomSource: rand.New(rand.NewSource(int64(time.Now().Nanosecond()))),
	}
}

// Get returns a random quote.
func (q *Generator) Get() string {
	return defaultQuotes[q.randomSource.Intn(len(defaultQuotes))]
}
