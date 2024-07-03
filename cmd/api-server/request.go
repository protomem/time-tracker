package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/protomem/time-tracker/internal/model"
)

const _customTimeLayout = "2006-01-02 15:04:05 MST"

func userIDFromRequest(r *http.Request) (model.ID, error) {
	id, err := strconv.Atoi(chi.URLParam(r, "userId"))
	return model.ID(id), err
}

func taskIDFromRequest(r *http.Request) (model.ID, error) {
	id, err := strconv.Atoi(chi.URLParam(r, "taskId"))
	return model.ID(id), err
}

func timeQueryParams(r *http.Request, key string, layout ...string) (time.Time, bool, error) {
	layout = append(layout, _customTimeLayout)
	val, ok := r.URL.Query().Get(key), r.URL.Query().Has(key)
	if !ok {
		return time.Time{}, false, nil
	}
	val = strings.TrimPrefix(val, "'")
	val = strings.TrimPrefix(val, "\"")
	val = strings.TrimSuffix(val, "'")
	val = strings.TrimSuffix(val, "\"")
	t, err := time.Parse(layout[0], val)
	return t, true, err
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
