package meme

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// QuoteClient provides functionality to call the Quote application.
type QuoteClient struct {
	baseURL string
}

// NewQuoteClient returns a new QuoteClient instance.
func NewQuoteClient(baseURL string) *QuoteClient {
	return &QuoteClient{
		baseURL: baseURL,
	}
}

// Get returns the random quote.
func (c *QuoteClient) Get() (string, error) {
	url := fmt.Sprintf("%s/quote", c.baseURL)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("while making HTTP call: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusOK:
	default:
		return "", fmt.Errorf("Got wrong status code. Expected: [%d], got: [%d] ", http.StatusOK, resp.StatusCode)
	}

	bodyRaw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("while reading HTTP response body: %w", err)
	}

	var dto struct {
		Quote string `json:"quote"`
	}
	if err = json.Unmarshal(bodyRaw, &dto); err != nil {
		return "", fmt.Errorf("while decoding HTTP response: %w", err)
	}

	return dto.Quote, nil
}
