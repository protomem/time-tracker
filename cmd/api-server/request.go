package main

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/protomem/time-tracker/internal/database"
	"github.com/protomem/time-tracker/internal/model"
)

const _customTimeLayout = "01-02-2006 15:04 MST"

const (
	_defaultPage     = 1
	_defaultPageSize = 10
)

func userIDFromRequest(r *http.Request) (model.ID, error) {
	id, err := strconv.ParseUint(chi.URLParam(r, "userId"), 10, 32)
	return model.ID(id), err
}

func taskIDFromRequest(r *http.Request) (model.ID, error) {
	id, err := strconv.ParseUint(chi.URLParam(r, "taskId"), 10, 32)
	return model.ID(id), err
}

func findOptionsFromRequest(r *http.Request) database.FindOptions {
	page := defaultUintQueryParams(r, "page", _defaultPage)
	pageSize := defaultUintQueryParams(r, "pageSize", _defaultPageSize)
	return database.FindOptions{
		Limit:  pageSize,
		Offset: (page - 1) * pageSize,
	}
}

func findUserFilterFromRequest(r *http.Request) database.FindUserFilter {
	return database.FindUserFilter{
		Name:           optionalStringQueryParams(r, "name"),
		Surname:        optionalStringQueryParams(r, "surname"),
		Patronymic:     optionalStringQueryParams(r, "patronymic"),
		Address:        optionalStringQueryParams(r, "address"),
		PassportSerie:  optionalIntQueryParams(r, "passportSerie"),
		PassportNumber: optionalIntQueryParams(r, "passportNumber"),
	}
}

func sessionTimelineOptionsFromRequest(r *http.Request) (database.SessionTimelineOptions, error) {
	opts := database.SessionTimelineOptions{}

	after, ok, err := timeQueryParams(r, "after")
	if err != nil {
		return database.SessionTimelineOptions{}, err
	}
	if ok {
		opts.After = &after
	}

	before, ok, err := timeQueryParams(r, "before")
	if err != nil {
		return database.SessionTimelineOptions{}, err
	}
	if ok {
		now := time.Now()
		if now.Before(before) {
			before = now
		}

		opts.Before = &before
	}

	return opts, nil
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
	return t, ok, err
}

func defaultUintQueryParams(r *http.Request, key string, def uint64) uint64 {
	val, ok := r.URL.Query().Get(key), r.URL.Query().Has(key)
	if !ok {
		return def
	}
	uintVal, err := strconv.ParseUint(val, 10, 64)
	if err != nil {
		return def
	}
	return uintVal
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
