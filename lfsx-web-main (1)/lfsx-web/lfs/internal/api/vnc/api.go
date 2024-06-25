package vnc

import (
	"fmt"
	"net/http"

	"gitea.hama.de/LFS/go-webserver/errors"
	"gitea.hama.de/LFS/go-webserver/response"
	"gitea.hama.de/LFS/lfsx-web/controller/pkg/utils"
	"github.com/go-chi/chi"
)

type Service interface {
	ChangeResoulution(width int, height int) error
	ChangeScaling(scaling int) error
	ChangeSwayScaling(scaling int) error
}

type ressource struct {
	service Service
}

func RegisterHandlers(r chi.Router, service Service) {
	res := ressource{service: service}

	r.Post("/vnc/resolution", res.ChangeResoulution)
	r.Post("/vnc/scale", res.ChangeScaling)
	r.Post("/vnc/scale/hard", res.ChangeScalingHard)
}

// ChangeResoulution applies the provided resolution for the virtual display
func (res ressource) ChangeResoulution(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Width  int `json:"width"`
		Height int `json:"height"`
	}

	// Get body
	utils.DecodeBody(&data, r)

	// Change resolution
	if err := res.service.ChangeResoulution(data.Width, data.Height); err != nil {
		response.WriteError(err, w, r)
	} else {
		response.WriteText("OK", 200, w)
	}
}

// ChangeScaling applies the provided, (fractional) scaling factor of the window manager.
// This does not prodcue clear text or layouts for the LFS.X!
func (res ressource) ChangeScaling(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Factor int `json:"factor"`
	}

	// Get body
	if _, err := utils.DecodeBody(&data, r); err != nil {
		errors.Write(w, err)
		return
	}

	// Validate factor
	if data.Factor < 50 {
		response.WriteText(fmt.Sprintf("Incorrect scalling factor given: %d. Expected a number like to 100(%%)", data.Factor), 500, w)
		return
	}

	// Change scaling
	if err := res.service.ChangeSwayScaling(data.Factor); err != nil {
		response.WriteText("OK", 200, w)
	} else {
		errors.Write(w, err)
	}

}

// ChangeScalingHard applies the provided, (fractional) scaling factor for gnome.
// Calling this endpoint results into a restart of the LFS.X
func (res ressource) ChangeScalingHard(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Factor int `json:"factor"`
	}

	// Get body
	utils.DecodeBody(&data, r)

	// Validate factor
	if data.Factor < 50 {
		response.WriteText("Incorrect scalling factor given. Expected a number like to 100(%)", 500, w)
	}

	// Change scaling
	if err := res.service.ChangeScaling(data.Factor); err != nil {
		response.WriteText(err.Error(), 207, w)
	} else {
		response.WriteText("OK", 200, w)
	}

}
