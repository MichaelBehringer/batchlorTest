package models

import (
	"os"
	"strings"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/pkg/utils"
)

// AppConfig contains generic configuration options for the app.
// Note that this file only contains sensitive or complex informations
// that cannot be exposed via environment variables
type AppConfig struct {
	Version string

	// Address on which the server should be listening on
	Address string

	// If the application should serve an LFS.X in production mode
	Production bool

	// The URL of the LFS endpoint to authenticate the user against.
	// This is used to validate the provided username and password
	// of the controller
	LfsServiceEndpoint string

	// The JWT key of the LFS service endpoint to decrypt the JWT token
	LfsJwtKey []byte

	// LFS Jwt Name
	LfsJwtName string

	// The name of the docker image that is used to start an LFS container.
	// Defaulting to @latest
	lfsImageName string
	// Read the name of the image from the specified file. This does override the
	// field "LfsImageName"
	lfsImageNameFile string

	// Development options
	DevConfig DevConfig
}

// The development config contains some options that are only needed
// during the development of this app
type DevConfig struct {

	// If the development server should be enabled
	DevServer bool

	// Port on which the development server should listen to
	DevServerPort int

	// Instead of getting the path to the VNC backend from kubernetes the local
	// path is used for ALL clusters
	VncAddress string

	// Instead of getting the path to the Guacamole backend from kubernetes the local
	// path is used for ALL clusters
	GuacamoleAddress string
}

// GetAppConfig gets all configuration options from the current environment variables.
// It panics if not all informations were provideded correctly because they are required
func GetAppConfig(version string) *AppConfig {

	// Apply logger configurations
	logger.SetGlobalLogger(logger.GetLoggerFromEnv(&logger.Logger{
		ColoredOutput: true,
		PrintLevel:    logger.LevelInfo,
		PrintSource:   true,
	}))

	// Initialize variables
	config := &AppConfig{}
	var err error

	// Get JWT key
	jwtKeyPath := utils.GetEnvString("APP_JWT_FILE", "./key.txt")
	config.LfsJwtKey, err = os.ReadFile(jwtKeyPath)
	if err != nil {
		logger.Fatal("Cannot read private JWT key from file %q: %s", jwtKeyPath, err)
	}

	// Get the address to listen on
	config.Address = utils.GetEnvString("APP_ADDRESS", ":4020")

	// Some other configuration
	config.Production = utils.GetEnvBool("APP_PRODUCTION", true)
	config.LfsServiceEndpoint = utils.RequireEnvString("APP_LFS_SERVICE_ENDPOINT")
	config.LfsJwtName = utils.GetEnvString("APP_LFS_SERVICE_ENDPOINT_JWT_NAME", "JWTAuthentication")
	config.lfsImageName = utils.GetEnvString("APP_LFS_IMAGE_NAME", utils.GetEnvString("APP_LFS_IMAGE_REGISTRY", "containers-next.hama.de/registry-hama-test/lfsx-web-lfs")+":"+version)
	config.lfsImageNameFile = utils.GetEnvString("APP_LFS_IMAGE_NAME_FILE", "")

	// Get development configs
	config.DevConfig.DevServer = utils.GetEnvBool("APP_DEV_USE_DEVSERVER", false)
	config.DevConfig.DevServerPort = utils.GetEnvInt("APP_DEV_SERVER_PORT", 5173)
	config.DevConfig.VncAddress = utils.GetEnvString("APP_DEV_VNC_ADDRESS", "")
	config.DevConfig.GuacamoleAddress = utils.GetEnvString("APP_DEV_GUACAMOL_ADDRESS", "")

	// Set version
	config.Version = version

	return config
}

// GetLfsImage returns the image name to use for the LFS.X container
func (c *AppConfig) GetLfsImage() string {
	// Read from file
	if c.lfsImageNameFile != "" {
		if content, err := os.ReadFile(c.lfsImageNameFile); err == nil {
			return strings.TrimSpace(string(content))
		} else {
			logger.Warning("Failed to read image tag of the LFS.X: %s", err)
		}
	}

	// Return the static value
	return c.lfsImageName
}

// GetLfsImageVersion returns the images version to use for the LFS.X container
func (c *AppConfig) GetLfsImageVersion() string {
	// Kubernetes only supports label that hava a maximum length of 63 characters
	maxLength := 60
	image := c.GetLfsImage()

	// Only return the version
	if index := strings.LastIndex(image, ":"); index != -1 && index < len(image) {
		return cutOff(image[index+1:], maxLength)
	}

	return cutOff(image, maxLength)
}

// cutOff returns a string that is cut off after the given amount of characters
func cutOff(val string, length int) string {
	if len(val) > length {
		return val[0:length]
	}

	return val
}
