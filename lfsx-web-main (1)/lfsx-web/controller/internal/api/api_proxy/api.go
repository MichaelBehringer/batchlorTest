package api_proxy

import (
	"fmt"
	"net/http"
	"strings"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/errors"
	"gitea.hama.de/LFS/go-webserver/response"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/lesismal/nbio/nbhttp/websocket"
)

type Service interface {
	IsUserConnected(user *models.User) bool
	ProxyLfsxRequest(user *models.User, response http.ResponseWriter, request *http.Request) error
	ProxyLfsxWebsocket(user *models.User, response http.ResponseWriter, request *http.Request) error
	ProxyHostRequest(user *models.User, response http.ResponseWriter, request *http.Request) error
}

type ressource struct {
	service Service
}

// RegisterHandlers register a endpoint that forwards all incoming requests to the LFS.X / Host
// endpoint and returns the response
func RegisterHandlers(r chi.Router, service Service) {
	res := ressource{service: service}

	r.Get("/connected", res.IsConnected)
	r.Get("/app/ws", res.onWebsocket)
	r.HandleFunc("/app/*", res.ProxyLfs)
	r.HandleFunc("/host/*", res.ProxyHost)
}

func (res ressource) ProxyLfs(w http.ResponseWriter, r *http.Request) {
	// Get the user of the request
	user := r.Context().Value(models.KeyUser).(*models.User)

	// Remove '/app' from path
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/api/app")

	// Proxy the request
	if err := res.service.ProxyLfsxRequest(user, w, r); err != nil {
		errors.Write(w, err)
	}
}

// ProxyHost is a static function that proxies the given request to the host endpoint
func ProxyHost(w http.ResponseWriter, r *http.Request, service Service) {
	res := ressource{service: service}
	res.ProxyHost(w, r)
}

func (res ressource) ProxyHost(w http.ResponseWriter, r *http.Request) {
	// Get the user of the request
	user := r.Context().Value(models.KeyUser).(*models.User)

	// Remove '/host' from path
	r.URL.Path = "/api" + strings.TrimPrefix(r.URL.Path, "/api/host")

	// Proxy the request
	if err := res.service.ProxyHostRequest(user, w, r); err != nil {
		errors.Write(w, err)
	}
}

func (res ressource) IsConnected(w http.ResponseWriter, r *http.Request) {
	// Get the user of the request
	user := r.Context().Value(models.KeyUser).(*models.User)
	isConnected := res.service.IsUserConnected(user)

	// Build response
	rtc := struct {
		Status string `json:"status"`
	}{
		Status: "connected",
	}
	if !isConnected {
		rtc.Status = "disconnected"
	}

	response.WriteJson(rtc, 200, w)
}

func (res ressource) onWebsocket(w http.ResponseWriter, r *http.Request) {

	// Get the user of the request
	user := r.Context().Value(models.KeyUser).(*models.User)

	// Proxy the request
	if err := res.service.ProxyLfsxWebsocket(user, w, r); err != nil {
		logger.Trc("Failed to proxy lfsx: %s", err)
		// Only return an error if its an own error.
		// All othe errors are already been written during the upgrade process
		if errResponse, ok := err.(errors.ErrorResponse); ok {
			// We need to upgrade the connection and send the error message over a WebSocket close message
			u := websocket.NewUpgrader()
			wsConn, err := u.Upgrade(w, r, nil)
			if err != nil {
				response.WriteError(err, w, r)
			} else {
				wsConn.WriteMessage(websocket.CloseMessage, []byte(fmt.Sprintf("%d: %s", errResponse.Status, errResponse.Message)))
				wsConn.Close()
			}
		}
	}
}
