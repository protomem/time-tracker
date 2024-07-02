package main

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/protomem/time-tracker/internal/model"
)

func userIDFromRequest(r *http.Request) (model.ID, error) {
	id, err := strconv.Atoi(chi.URLParam(r, "userId"))
	return model.ID(id), err
}
