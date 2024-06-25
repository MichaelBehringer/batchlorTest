package models

import (
	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/pkg/utils"
)

// AppConfig contains generic configuration options for the app
type AppConfig struct {
	Version string

	// Address on which the server should be listening on
	Address string
}

// GetAppConfig gets all configuration options from the current environment variables.
// It panics if not all informations were provideded correctly because they are required
func GetAppConfig(version string) *AppConfig {

	// Apply logger configurations
	logger.SetGlobalLogger(logger.GetLoggerFromEnv(&logger.Logger{
		ColoredOutput: true,
		PrintLevel:    logger.LevelDebug,
		PrintSource:   true,
	}))

	// Initialize variables
	return &AppConfig{
		Version: version,
		Address: utils.GetEnvString("APP_LFS_ADDRESS", ":4021"),
	}
}
