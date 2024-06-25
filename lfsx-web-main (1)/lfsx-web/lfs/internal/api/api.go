package api

import (
	"net/http"
	"os"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/response"
	"gitea.hama.de/LFS/go-webserver/webserver"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/api/kubernetes"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/api/vnc"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/lfs"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/models"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// Api contains dependencies of the programm
// that are needed from the API
type Api struct {
	Config     *models.AppConfig
	Lfs        *lfs.Lfs
	vncService *vnc.VncService
}

// Routes Setups and initializes all the api endpoints and registers the routes
func Routes(server *webserver.WebServer[Api]) http.Handler {
	router := chi.NewRouter()
	router.Use(middleware.RealIP, server.RecoverPanic, server.LogRequest, server.SecureHeaders)

	// API with shared dependencies
	api := server.Dependency

	// Register routes
	router.Route("/api", func(apiRouter chi.Router) {
		// All routes can be accessed without authentication
		api.routes(apiRouter)
	})

	return router
}

// routes returns a handler that mounts all "default"
// routes (with authentication) under the main API path
func (api *Api) routes(r chi.Router) {

	// Kubernetes specific endpoints
	kubernetes.RegisterHandlers(r, api.Config, api.Lfs)

	// VNC endpoints
	api.vncService = vnc.NewVncService(api.Lfs)
	vnc.RegisterHandlers(r, api.vncService)
	go api.vncService.StartUserConnectionsCheck()

	// Extra endpoints
	api.extras(r)
}

// extras registers special and small endpoints that aren't worth creating
// an own API file :) -> at a max. 5 lines of code each
func (api *Api) extras(r chi.Router) {

	// Stop this container
	r.Post("/stop", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received stop request from API. Leaving now....")

		// Stop the LFS.X
		if err := api.Lfs.Process.Process.Kill(); err != nil {
			response.WriteText("Failed to stop the LFS.X. Leaving anyway", 200, w)
		} else {
			response.WriteText("OK", 200, w)
		}

		// This app was the init command so the pod get's terminated without using the Kubernetes api
		os.Exit(0)
	})

	r.Post("/start", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("Received start command from API. Starting connectivity check now")
		api.vncService.WasConnected.Store(true)
	})
}
