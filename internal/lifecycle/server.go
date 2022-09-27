package lifecycle

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/httpsrv"
)

func NewServer(log logrus.FieldLogger, k8sCli kubernetes.Interface, cfg config.LifecycleServer) *httpsrv.Server {
	addr := fmt.Sprintf(":%d", cfg.Port)
	router := mux.NewRouter()
	restartHandler := newRestartHandler(log, k8sCli, cfg)
	router.HandleFunc("/restart", restartHandler)
	return httpsrv.New(log, addr, router)
}
