package source

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/httpx"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const (
	sourceNameVarName         = "sourceName"
	incomingWebhookPathPrefix = "sources/v1"
)

// IncomingWebhookData holds information about incoming webhook.
type IncomingWebhookData struct {
	inClusterBaseURL string
}

func (w IncomingWebhookData) FullURLForSource(sourceName string) string {
	return fmt.Sprintf("%s/%s/%s", w.inClusterBaseURL, incomingWebhookPathPrefix, sourceName)
}

// NewIncomingWebhookServer creates a new HTTP server for incoming webhooks.
func NewIncomingWebhookServer(log logrus.FieldLogger, cfg *config.Config, dispatcher *Dispatcher, startedSources map[string][]StartedSource) *httpx.Server {
	addr := fmt.Sprintf(":%d", cfg.Plugins.IncomingWebhook.Port)
	router := incomingWebhookRouter(log, cfg, dispatcher, startedSources)

	log.Info("Starting server on %q...", addr)
	return httpx.NewServer(log, addr, router)
}

func incomingWebhookRouter(log logrus.FieldLogger, cfg *config.Config, dispatcher *Dispatcher, startedSources map[string][]StartedSource) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc(fmt.Sprintf("/%s/{%s}", incomingWebhookPathPrefix, sourceNameVarName), func(writer http.ResponseWriter, request *http.Request) {
		sourceName, ok := mux.Vars(request)[sourceNameVarName]
		if !ok {
			writeJSONError(log, writer, "Source name in path is required", http.StatusBadRequest)
			return
		}
		logger := log.WithFields(logrus.Fields{
			"sourceName": sourceName,
		})
		logger.Debugf("Handling incoming webhook request...")

		sourcePlugins, ok := startedSources[sourceName]
		if !ok {
			writeJSONError(log, writer, fmt.Sprintf("source %q not found", sourceName), http.StatusNotFound)
			return
		}

		payload, err := io.ReadAll(request.Body)
		if err != nil {
			writeJSONError(log, writer, fmt.Sprintf("while reading request body: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		defer request.Body.Close()

		multiErr := multierror.New()
		for _, src := range sourcePlugins {
			logger.WithFields(logrus.Fields{
				"pluginName":               src.PluginName,
				"isInteractivitySupported": src.IsInteractivitySupported,
			}).Debug("Dispatching message...")

			err := dispatcher.DispatchExternalRequest(ExternalRequestDispatch{
				PluginDispatch: PluginDispatch{
					ctx:                      context.Background(),
					sourceName:               sourceName,
					sourceDisplayName:        src.SourceDisplayName,
					pluginName:               src.PluginName,
					pluginConfig:             src.PluginConfig,
					isInteractivitySupported: src.IsInteractivitySupported,
					cfg:                      cfg,
					pluginContext:            config.PluginContext{},
					incomingWebhook: IncomingWebhookData{
						inClusterBaseURL: cfg.Plugins.IncomingWebhook.InClusterBaseURL,
					},
				},
				payload: payload,
			})
			if err != nil {
				multiErr = multierror.Append(multiErr, err)
			}
		}

		if multiErr.ErrorOrNil() != nil {
			wrappedErr := fmt.Errorf("while dispatching external request: %w", multiErr)
			writeJSONError(log, writer, wrappedErr.Error(), http.StatusInternalServerError)
			return
		}

		writeJSONSuccess(log, writer)
	}).Methods(http.MethodPost)
	return router
}

func writeJSONError(log logrus.FieldLogger, w http.ResponseWriter, errMsg string, code int) {
	response := struct {
		Error string `json:"error"`
	}{
		Error: errMsg,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		log.Errorf("while writing error response: %s", err.Error())
	}
}

func writeJSONSuccess(log logrus.FieldLogger, w http.ResponseWriter) {
	response := struct {
		Success bool `json:"success"`
	}{
		Success: true,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(&response)
	if err != nil {
		log.Errorf("while writing success response: %s", err.Error())
	}
}
