package kubernetes

import (
	"net/http"
	"syscall"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/errors"
	"gitea.hama.de/LFS/go-webserver/response"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/lfs"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/models"
	"github.com/go-chi/chi"
)

type ressource struct {
	config *models.AppConfig
	lfs    *lfs.Lfs
}

// RegisterHandlers register endpoints that are needed for kubernetes
// to check the current status of the pod
func RegisterHandlers(r chi.Router, config *models.AppConfig, lfs *lfs.Lfs) {
	res := ressource{config: config, lfs: lfs}

	r.Get("/healthz", res.HealthCheck)
	r.Get("/readyz", res.ReadinessCheck)
}

func (res *ressource) HealthCheck(w http.ResponseWriter, r *http.Request) {

	// Send a status code 0 to the LFS process to
	if err := res.lfs.Process.Process.Signal(syscall.Signal(0)); err == nil {
		response.WriteText("OK", 200, w)
	} else {
		logger.Error("LFS process is dead: %s", err)
		response.WriteError(errors.NewError(err.Error(), 400), w, r)
	}
}

func (res *ressource) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	response.WriteText("OK", 200, w)
}
