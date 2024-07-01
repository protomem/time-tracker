package main

import (
	"net/http"

	"github.com/protomem/time-tracker/internal/response"
)

// Handle Status
// @Summary Server Status
// @Description Check if the server is up and running
// @Tags api
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /status [get]
func (app *application) handleStatus(w http.ResponseWriter, r *http.Request) {
	if err := response.JSON(w, http.StatusOK, response.JSONObject{"status": "OK"}); err != nil {
		app.serverError(w, r, err)
	}
}
