package api

import (
	"encoding/json"
	"io/fs"
	"net/http"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/frontend"
	"gitea.hama.de/LFS/go-webserver/webserver"
	"gitea.hama.de/LFS/lfsx-web/controller/web"
	"github.com/go-chi/chi/v5"
)

// frontendVariables contains configuration options
// that are exposed to the web interface
type frontendVariables struct {
	Prod bool `json:"prod"`
}

// setupFrontend configures the frontend of the application
func (api *Api) setupFrontend(server *webserver.WebServer[Api], router *chi.Mux) {
	// Variables to inject for the web interface
	config, err := json.Marshal(frontendVariables{Prod: server.Dependency.Config.Production})
	if err != nil {
		logger.Fatal("Failed to marshal struct to json: %s", err)
	}
	variableMap := map[string]string{
		"Config": string(config),
	}

	// Setup frontend
	fr := frontend.Frontend{
		Logger: server.Logger,
		Config: frontend.FrontendConfig{
			DevPort:         api.Config.DevConfig.DevServerPort,
			DevServer:       api.Config.DevConfig.DevServer,
			CompiledSources: web.FrontendFiles,
			Variables:       variableMap,
			Title:           "LFS.X",
			WebPath:         "./controller/web/app",
			Favicon:         "/static/favicon.png",
			FaviconType:     "image/png",
		},
		WebConfig: server.Config,
	}
	fr.SetupServer(router)

	// Serve static files
	if staticFolder, err := fs.Sub(web.StaticFiles, "app/src/static"); err != nil {
		logger.Error("Cannot access the embedded directory 'static': %s", err)
	} else {
		frontend.FileServer(router, "/static", http.FS(staticFolder))
	}
}
