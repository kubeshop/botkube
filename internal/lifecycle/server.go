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
	reloadHandler := newReloadHandler(log, k8sCli, cfg)
	router.HandleFunc("/reload", reloadHandler)
	return httpsrv.New(log, addr, router)
}
