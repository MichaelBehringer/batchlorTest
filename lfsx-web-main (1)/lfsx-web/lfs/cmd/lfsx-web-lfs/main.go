package main

import (
	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/webserver"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/api"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/lfs"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/models"
)

var version string

func main() {
	defer logger.CloseFile()

	// Apply gneric configuration options
	conf := models.GetAppConfig(version)

	// Start the LFS
	lfs, err := lfs.StartLfs(conf)
	if err != nil {
		logger.Fatal("Failed to start the LFS: %s", err)
	}

	// Build the web app
	webApp := webserver.WebServer[api.Api]{
		Logger: logger.GetGlobalLogger(),
		Dependency: api.Api{
			Config: conf,
			Lfs:    lfs,
		},
		Config: &webserver.WebConfig{
			Address: conf.Address,
		},
	}
	webApp.Setup(api.Routes)

	// Start the application
	logger.Info("Started up the controller (v%s)", conf.Version)
	webApp.Start()
}
