package kubernetes

import (
	"net/http"

	"gitea.hama.de/LFS/go-webserver/response"
	"github.com/go-chi/chi/v5"
)

type ressource struct{}

// RegisterHandlers register endpoints that are needed for kubernetes
// to check the current status of the pod.
func RegisterHandlers(r chi.Router) {
	res := ressource{}

	r.Get("/healthz", res.HealthCheck)
	r.Get("/readyz", res.ReadinessCheck)
}

func (res *ressource) HealthCheck(w http.ResponseWriter, r *http.Request) {
	response.WriteText("OK", 200, w)
}

func (res *ressource) ReadinessCheck(w http.ResponseWriter, r *http.Request) {
	response.WriteText("OK", 200, w)
}
