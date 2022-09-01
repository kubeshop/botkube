package meme

import (
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Handler is a wrapper to utilize meme generator on HTTP service.
type Handler struct {
	memeGenerator *Generator
	log           logrus.FieldLogger
}

// NewHandler returns a new Handler instance.
func NewHandler(log logrus.FieldLogger, memGen *Generator) *Handler {
	return &Handler{
		memeGenerator: memGen,
		log:           log,
	}
}

// GetRandomMemeHandler handles the meme request.
func (h *Handler) GetRandomMemeHandler(rw http.ResponseWriter, _ *http.Request) {
	reader, err := h.memeGenerator.Get()
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		h.log.Error("while getting random meme: %s", err)
		return
	}

	_, err = io.Copy(rw, reader)
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		rw.Write([]byte(err.Error()))
		h.log.Error("while copying random meme: %s", err)
		return
	}
}
