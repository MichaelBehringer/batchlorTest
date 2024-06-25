// The lfs package is responsible to controle the started
// LFSX instance within the container
package lfs

import (
	"context"
	"os"
	"os/exec"
	"syscall"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/lfsx-web/controller/pkg/utils"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/models"
)

// Lfs contains and controlls the LFS.X process that is currently
// running
type Lfs struct {
	// The started lfs process
	Process *exec.Cmd

	// App configuration
	config *models.AppConfig
}

// StartLfs boots the LFS up inside the container as a sub process
func StartLfs(config *models.AppConfig) (*Lfs, error) {

	// Create process
	lfs := exec.Command(
		utils.GetEnvString("APP_LFS_PROC_PATH", "/opt/lfsx/lfsx"),
		"-data", utils.GetEnvString("APP_LFS_PROC_DATA", "/opt/lfs-user"),
		"-lfsServicesBaseUrl", utils.GetEnvString("APP_LFS_SERVICE_ENDPOINT", "https://webapi.hama.com/lfstest"),
		"-lfsConf", utils.GetEnvString("APP_LFS_CONFIG", "/opt/lfs-user/config-dev"),
	)
	// Set process group id for child processes so we can kill them all from the parent
	lfs.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Pipe output to this terminal
	lfs.Stdout = os.Stdout
	lfs.Stderr = os.Stderr

	l := &Lfs{
		Process: lfs,
		config:  config,
	}

	// Check for errors
	go func() {
		ctx := context.Background()

		// Start the process
		if err := lfs.Run(); err != nil {
			logger.Info("Executed the LFS.X with an error: %s", err.Error())

			if e, ok := err.(*exec.ExitError); ok && ctx.Err() == nil {
				if e.ProcessState != nil && e.ProcessState.ExitCode() != 0 {
					// Restart the LFS. This could lead to a race condition because we reassign the pointer, but it's unlikely...
					logger.Info("Restarting the LFS.X because it was exited with a return code != 0: %d", e.ProcessState.ExitCode())
					newL, _ := StartLfs(config)
					*l = *newL
				}
			} else {
				logger.Info("Not trying to restart the LFS.X")
			}
		} else if ctx.Err() == nil {
			// Restart the LFS. This could lead to a race condition because we reassign the pointer, but it's unlikely...
			logger.Info("Exited the LFS.X but the main program / container should not be termined. Restarting...")
			newL, _ := StartLfs(config)
			*l = *newL
		}
	}()

	return l, nil
}
