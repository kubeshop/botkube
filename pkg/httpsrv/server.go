package httpsrv

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Server provides functionality to start HTTP server with a cancelable context.
type Server struct {
	srv *http.Server
	log logrus.FieldLogger
}

// New creates a new HTTP server.
func New(log logrus.FieldLogger, addr string, mux *http.ServeMux) *Server {
	return &Server{
		srv: &http.Server{Addr: addr, Handler: mux},
		log: log,
	}
}

// Serve starts the HTTP server and blocks unil the channel is closed or an error occurs.
func (s *Server) Serve(ctx context.Context) error {
	go func() {
		<-ctx.Done()
		s.log.Info("Context canceled. Finishing...")
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
