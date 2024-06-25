package vnc

import (
	"net/http"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/errors"
	"gitea.hama.de/LFS/go-webserver/response"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"gitea.hama.de/LFS/lfsx-web/controller/pkg/utils"
	"github.com/go-chi/chi/v5"
)

type Service interface {
	Proxy(w http.ResponseWriter, r *http.Request, user *models.User, useGuacamole bool, vncSettings VncConnectionSettings) error
	Probe(user *models.User, vncSettings VncConnectionSettings) error
}

type ressource struct {
	service Service
}

// VncConnectionRequest contains user specific settings to apply for the
// VNC session
type VncConnectionSettings struct {
	// Scaling factor of the application based on 100%
	Scaling int
}

func RegisterHandlers(r chi.Router, service Service) {
	res := ressource{service: service}

	r.Get("/vnc/ws", res.onWebsocket)
	r.Get("/vnc/ws/probe", res.probeConnection)
}

func (res ressource) onWebsocket(w http.ResponseWriter, r *http.Request) {

	// Get the user of the request
	user := r.Context().Value(models.KeyUser).(*models.User)

	// Get VNC options
	settings := res.getVncSettings(r)

	// Proxy the request
	if err := res.service.Proxy(w, r, user, r.URL.Query().Get("useGuacamole") == "true", settings); err != nil {
		logger.Debug("Received error from proxy: %s", err)

		// Only return an error if its an own error.
		// All othe errors are already been written during the upgrade process
		if _, ok := err.(errors.ErrorResponse); ok {
			errors.Write(w, err)
		}
	}
}

func (res ressource) probeConnection(w http.ResponseWriter, r *http.Request) {
	// Get the user of the request
	user := r.Context().Value(models.KeyUser).(*models.User)

	// Get VNC options
	settings := res.getVncSettings(r)

	if err := res.service.Probe(user, settings); err == nil {
		response.WriteText("Ok", 200, w)
	} else {
		errors.Write(w, err)
	}
}

func (res ressource) getVncSettings(r *http.Request) VncConnectionSettings {
	return VncConnectionSettings{
		Scaling: utils.GetQueryValueInt("scale", 100, r),
	}
}
