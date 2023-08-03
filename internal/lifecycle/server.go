package lifecycle

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/httpx"
	"github.com/kubeshop/botkube/pkg/config"
)

type Restarter interface {
	Do(ctx context.Context) error
}

// SendMessageFn defines a function which sends a given message.
type SendMessageFn func(msg string) error

// NewServer creates a new httpsrv.Server that exposes lifecycle methods as HTTP endpoints.
func NewServer(log logrus.FieldLogger, cfg config.LifecycleServer, restarter Restarter) *httpx.Server {
	addr := fmt.Sprintf(":%d", cfg.Port)
	router := mux.NewRouter()
	reloadHandler := newReloadHandler(log, restarter)
	router.HandleFunc("/reload", reloadHandler)
	return httpx.NewServer(log, addr, router)
}

func newReloadHandler(log logrus.FieldLogger, restarter Restarter) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		log.Debug("Reload handler called. Executing restart...")

		err := restarter.Do(request.Context())
		if err != nil {
			errMsg := fmt.Sprintf("while restarting the Deployment: %s", err.Error())
			log.Error(errMsg)
			http.Error(writer, errMsg, http.StatusInternalServerError)
		}

		writer.WriteHeader(http.StatusOK)
		_, err = writer.Write([]byte("Deployment restarted successfully."))
		if err != nil {
			log.Errorf("while writing success response: %s", err.Error())
		}
	}
}
