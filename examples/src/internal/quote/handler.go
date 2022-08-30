package quote

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Handler is a wrapper to utilize quote generator on HTTP service.
type Handler struct {
	quoteProvider *Generator
	log           logrus.FieldLogger
}

// NewHandler returns a new Handler instance.
func NewHandler(log logrus.FieldLogger, quoteProvider *Generator) *Handler {
	return &Handler{
		quoteProvider: quoteProvider,
		log:           log,
	}
}

// GetRandomQuoteHandler handles the quote request.
func (h *Handler) GetRandomQuoteHandler(rw http.ResponseWriter, _ *http.Request) {
	dto := dto{Quote: h.quoteProvider.Get()}

	err := h.Encode(rw, dto)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		h.log.Error("while sending random quote: %s", err)
	}
}

// Encode encodes the given object to json format and writes it to given ResponseWriter
func (h *Handler) Encode(rw http.ResponseWriter, v interface{}) error {
	rw.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(rw).Encode(v)
}

type dto struct {
	Quote string `json:"quote"`
}
