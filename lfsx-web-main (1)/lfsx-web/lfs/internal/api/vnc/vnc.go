package vnc

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync/atomic"
	"syscall"
	"time"

	"gitea.hama.de/LFS/go-logger"
	"gitea.hama.de/LFS/go-webserver/errors"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/lfs"
	"gitea.hama.de/LFS/lfsx-web/lfs/internal/models"
)

// Duration in hours after which this pod should be termineted when
// no user was connected
const NO_USER_CONNECTION_TIME = 48

// VncService controlls options that are relevant
// for the VNC display output and must be executed
// in the context of swayvnc
type VncService struct {
	// Name of the virtual display in which the LFS.X is running
	DisplayName string

	// If the controller was already connected in the past
	WasConnected atomic.Bool

	// The date of the application bootup
	Uptime time.Time

	// A list of predefined scaling properties indexed by the scaling factor based on 100%
	ScalingModes map[int]models.Scaling

	// LFS instance
	lfs *lfs.Lfs
}

// NewVncService constructs a new VNC Service to manage the display output
func NewVncService(lfs *lfs.Lfs) *VncService {
	return &VncService{
		DisplayName: "HEADLESS-1",
		Uptime:      time.Now(),
		ScalingModes: map[int]models.Scaling{
			100: {Scaling: 100, ScalingFont: 100, CursorSize: 24},
			125: {Scaling: 100, ScalingFont: 125, CursorSize: 24},
			150: {Scaling: 100, ScalingFont: 150, CursorSize: 32},
			175: {Scaling: 100, ScalingFont: 175, CursorSize: 32},
			200: {Scaling: 200, ScalingFont: 100, CursorSize: 24},
		},
		lfs: lfs,
	}
}

// ChangeResoulution changes the displayed resoulution
// for the virtual display in which the LFS.X is running
func (v *VncService) ChangeResoulution(width int, height int) error {
	cmd := exec.Command("swaymsg", "output", v.DisplayName, "pos", "0", "0", "res", fmt.Sprintf("%dx%d", width, height))

	output, rtc, err := v.execute(cmd)
	if err != nil {
		logger.Warning("Failed to change the resoulution: %s (%d)", output, rtc)
		return errors.NewError("Failed to change the resoulution", 500)
	}

	return nil
}

// ChangeScaling applies the provided scaling factor.
// This does always restart the application
// in order to apply the scaling.
func (v *VncService) ChangeScaling(scaling int) error {

	// We dont't change the scaling after a user was connected (he has to delete the
	// pod and start a new one)
	if _, err := os.ReadFile("/home/oracle/.lfsx-user"); err == nil {
		return fmt.Errorf("lfs.x was already started for the user")
	}

	// Try to get a supported scaling factor (optimize font rendering)
	if res, found := v.ScalingModes[scaling]; found {
		gtkI := "org.gnome.desktop.interface"

		// Apply gtk specific functions. Changing the text and cursor size has the same effect as
		// setting the env variable "GDK_DPI_SCALE "
		if o, c, err := v.execute(exec.Command("gsettings", "set", gtkI, "text-scaling-factor", fmt.Sprintf("%.2f", float32(res.ScalingFont)/100.0))); err != nil {
			logger.Error("Failed to set text-scaling-factor (%d): %s", c, err)
			logger.Debug(o)
		}
		if o, c, err := v.execute(exec.Command("gsettings", "set", gtkI, "cursor-size", fmt.Sprintf("%d", res.CursorSize))); err != nil {
			logger.Error("Failed to set cursor-size (%d): %s", c, err)
			logger.Debug(o)
		}
		v.ChangeSwayScaling(res.Scaling)
	} else {
		v.ChangeSwayScaling(scaling)
	}

	// Stop an already running LFS.X instance
	if err := syscall.Kill(-v.lfs.Process.Process.Pid, syscall.SIGTERM); err != nil {
		logger.Warning("Failed to kill the LFS.X process to update display scaling: %s", err)
	}

	return nil
}

// ChangeSwayScaling applies the provided scaling factor for sway (whole WM).
// This does not require any restart of the application.
// Fonts could be blurry for apps that do not support the fractional scalling
// protocl of wayland (expect upscaling 2x).
// This has the same effect as setting the env variable "GDK_SCALE" for GDK apps.
// If you applied a gtk scaling factor before, this function will scale based on that gtk scaling factor.
// This may be no the behaviour you expect!
func (v *VncService) ChangeSwayScaling(scaling int) error {
	output, rtc, err := v.execute(exec.Command("swaymsg", "output", v.DisplayName, "scale", fmt.Sprintf("%.2f", float32(scaling)/100.0)))
	if err != nil {
		logger.Error("Failed to apply scaling for sway: %s (%d)", output, rtc)
		return errors.NewError("Failed to change scaling", 500)
	}

	return nil
}

// StartUserConnectionsCheck fetches the number of connected users
// repeatedly and exists the programm after a timeout of 5 minutes when no user
// is connected.
//
// This method does block infinite until the program is exited :)
func (v *VncService) StartUserConnectionsCheck() {

	// How often no user was connected. This counter will be reset after a user is connected again
	noUsersCount := 0

	for {
		time.Sleep(30 * time.Second)
		cmd := exec.Command("bash", "-c", "netstat -anpt | grep '5910' | grep -E 'ESTABLISHED \\d{2,}/wayvnc' | grep -c 'tcp'")

		output, rtc, err := v.execute(cmd)
		if err != nil {
			// In go an error is also returned, when the status code <> 0 -> only log error when it was no such error
			if _, ok := err.(*exec.ExitError); !ok {
				logger.Warning("Failed to fetch the number of connected users: %s (%d)", output, rtc)
				continue
			}
		}

		// Check if user is connected. Grep will return "0" when users > 0, otherwise "1"
		if rtc == 1 && v.WasConnected.Load() {
			noUsersCount++
			logger.Trc("No user is connected. Counter value is %d", noUsersCount)
		} else {
			noUsersCount = 0

			// Store the connection flag
			if rtc == 0 && !v.WasConnected.Load() {
				v.WasConnected.Store(true)
			}

			// Check if the no user was connected for a time > 24h. In such a case the pod will be terminated
			if time.Since(v.Uptime).Hours() >= NO_USER_CONNECTION_TIME {
				logger.Info("Connection time limit of %d hours reached", NO_USER_CONNECTION_TIME)
				os.Exit(0)
			}
		}

		// Maximum limit reachted
		if noUsersCount > 10 {
			logger.Info("Stopping container because no user was connected in the last 5 minutes")

			// This app was the init command so the pod get's terminated without using the Kubernetes api
			os.Exit(0)
		}
	}
}

// execute executes the given command and returns the combined
// stdout and stderr and the return code
func (v *VncService) execute(cmd *exec.Cmd) (output string, returnCode int, err error) {
	// Combine stdout and stderr
	cmdReader, err := cmd.StdoutPipe()
	if err != nil {
		return "", 0, err
	}

	cmd.Stderr = cmd.Stdout
	defer cmdReader.Close()

	// Function to read the combined output. The lock is used to make sure that the goroutines
	// executes before the command line is run
	go func() {
		outCombined, err := io.ReadAll(cmdReader)
		if err != nil {
			logger.Warning("Failed to read output from program: %s", err)
		}
		output = string(outCombined)
	}()

	// Execute it but wait some time before io.ReadAll cicks in. Timeouts are not cool, but im not aware of a clean solution...
	time.Sleep(8 * time.Millisecond)
	err = cmd.Run()

	// If a non zero return code was returned, an error is returned in go
	if err != nil {
		if werr, ok := err.(*exec.ExitError); ok {
			returnCode = werr.ExitCode()
		} else {
			return "", -1, err
		}
	}

	return
}
