package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/response"
	"gitea.hama.de/LFS/go-webserver/webserver"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/api/api_proxy"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/api/kubernetes"
	vnc "gitea.hama.de/LFS/lfsx-web/controller/internal/api/vnc_proxy"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/kuber"
	"gitea.hama.de/LFS/lfsx-web/controller/internal/models"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Api contains dependencies of the programm
// that are needed from the API
type Api struct {
	Config *models.AppConfig

	// VNC Service used for proxy the logout request
	vncService *vnc.VncProxy
}

// Routes Setups and initializes all the api endpoints and registers the routes
func Routes(server *webserver.WebServer[Api]) http.Handler {
	router := chi.NewRouter()

	// API with shared dependencies
	api := server.Dependency

	router.Use(middleware.RealIP, server.RecoverPanic, server.LogRequest, api.SecureHeaders /*, server.SecureHeaders*/)

	// Register routes
	router.Route("/api", func(apiRouter chi.Router) {

		// Routes with authentication
		apiRouter.Group(func(auth chi.Router) {
			auth.Use(api.AuthenticationMiddleware)

			// Apply default routes
			api.routes(auth)
		})

		// Routes with authentication "validation"
		apiRouter.Group(func(authVal chi.Router) {
			authVal.Use(api.AuthenticationMiddleware)

			authVal.Get("/isAuthenticated", api.queryAuthentication)
			authVal.Post("/logout", api.logout)
		})

		// Routes without authentication
		apiRouter.Group(func(noAuth chi.Router) {
			noAuth.Post("/login", api.login)

			// Register kubernetes health endpoints
			kubernetes.RegisterHandlers(noAuth)
		})
	})

	api.setupFrontend(server, router)

	return router
}

// @TODO add secure headers to request to allow websocket:  connect-src 'self' ws:;
// Zudem sollten die SecureHeaders nochmals überarbeitet weren :)
func (api *Api) SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			//"default-src 'self' localhost:*; style-src 'self' fonts.googleapis.com localhost:*; font-src fonts.gstatic.com")
			"default-src 'self'; script-src 'self' localhost:5173 'unsafe-inline'; connect-src 'self' ws: wss: localhost:5173; img-src * data: blob: 'unsafe-inline'; frame-src *; style-src 'self' localhost:5173 'unsafe-inline';")
		w.Header().Set("Referrer-Policy", "origin-when-cross-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "0")

		// Add cors header for QA-Portal
		if api.allowQaOrigin(w, r) {
			return
		}

		next.ServeHTTP(w, r)
	})
}

// allowQaOrigin adds an 'Access-Control-Allow-Origin' header if the
// requests origin matches one of the QA Domains. Because only one origin can
// be specified it's made conditional.
// It returns true if next.ServeHTTP should not be called (OPTIONS request)
func (api *Api) allowQaOrigin(w http.ResponseWriter, r *http.Request) bool {
	origin := r.Header.Get("Origin")

	// Not a request from a Web Browser
	if origin == "" {
		return false
	}
	isAllowed := false

	// From test systems
	if api.Config.Production {
		isAllowed = origin == "https://qa.hama.com"
	} else {
		isAllowed = origin == "https://qa-test.hama.com" || origin == "https://qa-rc.hama.com" || origin == "http://localhost:8081"
	}

	// Set header
	if isAllowed {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Db")

		// If it's an option request to the LFS.X allow it (without authentication middleware)
		if r.Method == "OPTIONS" && strings.HasPrefix(r.URL.Path, "/api/app/") {
			response.WriteText("OK", 200, w)
			return true
		}
	}

	return false
}

// routes returns a handler that mounts all "default"
// routes (with authentication) under the main API path
func (api *Api) routes(r chi.Router) {

	// Shared kubernetes client
	kuber, err := kuber.NewKuber(api.Config)
	if err != nil {
		logger.Fatal("Failed to create kubernetes client: %s", err)
	}

	// Start generic tasks
	api.startTasks(kuber)

	// VNC endpoints handling the WebSocket connection
	vncService, err := vnc.NewVncProxy(context.Background(), kuber, api.Config)
	if err != nil {
		logger.Fatal(err.Error())
	}
	api.vncService = vncService
	vnc.RegisterHandlers(r, vncService)

	// Register proxy endpoints
	api_proxy.RegisterHandlers(r, vncService)
}

// startTasks runs generic kubernetes tasks that are performed
// periodically / at startup.
// This method does not block. The tasks are only started
func (api *Api) startTasks(kuber *kuber.Kuber) {

	// Start a new thread to remove completed jobs
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		context := context.Background()
		lastImageVersion := api.Config.GetLfsImageVersion()

		for {
			select {
			case <-ticker.C:
				if err := kuber.DeleteCompletedLfsPods(); err != nil {
					logger.Warning("Failed to delete completed pods: %s", err)
				}

				// Check if image of lFS was changed
				if lastImageVersion != api.Config.GetLfsImageVersion() {
					logger.Info("Changed image version of the LFS.X: %s", api.Config.GetLfsImageVersion())
					lastImageVersion = api.Config.GetLfsImageVersion()
					api.createAtLeastOneJob(kuber)
				}
			case <-context.Done():
				logger.Info("Stopped looking for completed pods")
				ticker.Stop()
				return
			}
		}
	}()

	// The initial pulling and creation of a container (image) takes long. So create at least one pod at start
	go func() {
		api.createAtLeastOneJob(kuber)
	}()
}

// createAtLeastOneJob checks if at least one job is created for the
// current image version.
// If not, a single placeholder job is created
func (api *Api) createAtLeastOneJob(kuber *kuber.Kuber) {
	jobs, err := kuber.GetPlaceholders()
	if err == nil && len(jobs.Items) == 0 {
		if _, err := kuber.CreatePlaceholderJob(); err != nil {
			logger.Warning("Failed to create placeholders on startup / on image change")
		}
	} else if err != nil {
		logger.Debug("Failed to create placeholders: %s", err)
	} else {
		logger.Debug("No creation of placeholders is required (already started %d pods)", len(jobs.Items))
	}
}
