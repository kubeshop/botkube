package health

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kubeshop/botkube/internal/httpx"
	"github.com/sirupsen/logrus"
)

const (
	healthEndpointName = "/healthz"
)

type Checker struct {
	applicationStarted bool
}

func NewChecker() Checker {
	return Checker{
		applicationStarted: false,
	}
}

func (h *Checker) MarkAsReady() {
	h.applicationStarted = true
}

func (h *Checker) IsReady() bool {
	return h.applicationStarted
}

func (h *Checker) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	if h.IsReady() {
		resp.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(resp, "ok")
	} else {
		resp.WriteHeader(http.StatusServiceUnavailable)
		_, _ = fmt.Fprint(resp, "unavailable")
	}
}

func (h *Checker) NewServer(log logrus.FieldLogger, port string) *httpx.Server {
	addr := fmt.Sprintf(":%s", port)
	router := mux.NewRouter()
	router.Handle(healthEndpointName, h)
	return httpx.NewServer(log, addr, router)
}
