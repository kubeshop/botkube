package httpx

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const readHeaderTimeout = 1 * time.Minute

// Server provides functionality to start HTTP server with a cancelable context.
type Server struct {
	srv *http.Server
	log logrus.FieldLogger
}

// NewServer creates a new HTTP server.
func NewServer(log logrus.FieldLogger, addr string, handler http.Handler) *Server {
	return &Server{
		srv: &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: readHeaderTimeout},
		log: log,
	}
}

// Serve starts the HTTP server and blocks unil the channel is closed or an error occurs.
func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.log.Info("Shutdown requested. Finishing...")
		if err := s.srv.Shutdown(context.Background()); err != nil {
			s.log.Error("while shutting down server: %w", err)
		}
	}()

	s.log.Infof("Starting server on address %q", s.srv.Addr)
	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("while starting server: %w", err)
	}

	return nil
}
