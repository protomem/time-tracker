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

func taskIDFromRequest(r *http.Request) (model.ID, error) {
	id, err := strconv.Atoi(chi.URLParam(r, "taskId"))
	return model.ID(id), err
}

func stringQueryParams(r *http.Request, key string) (string, bool) {
	val, ok := r.URL.Query().Get(key), r.URL.Query().Has(key)
	return val, ok
}

func defaultStringQueryParams(r *http.Request, key string, def string) string {
	val, ok := r.URL.Query().Get(key), r.URL.Query().Has(key)
	if !ok {
		return def
	}
	return val
}

func defaultIntQueryParams(r *http.Request, key string, def int) int {
	val, ok := r.URL.Query().Get(key), r.URL.Query().Has(key)
	if !ok {
		return def
	}
	i, err := strconv.Atoi(val)
	if err != nil {
		return def
	}
	return i
}

func optionalStringQueryParams(r *http.Request, key string) *string {
	ref := new(string)
	val, ok := r.URL.Query().Get(key), r.URL.Query().Has(key)
	if !ok {
		return nil
	}
	*ref = val
	return ref
}

func optionalIntQueryParams(r *http.Request, key string) *int {
	ref := new(int)
	val, ok := r.URL.Query().Get(key), r.URL.Query().Has(key)
	if !ok {
		return nil
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return nil
	}
	*ref = intVal
	return ref
}
