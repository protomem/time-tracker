package main

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/protomem/time-tracker/internal/ctxstore"
	"github.com/protomem/time-tracker/internal/response"
	"github.com/rs/cors"

	"github.com/tomasen/realip"
)

const _traceIDKey = ctxstore.Key("traceId")

func (app *application) traceID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tid := genTraceID()
		ctx := ctxstore.With(r.Context(), _traceIDKey, tid)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err != nil {
				app.serverError(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (app *application) logAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mw := response.NewMetricsResponseWriter(w)
		next.ServeHTTP(mw, r)

		var (
			ip     = realip.FromRequest(r)
			method = r.Method
			url    = r.URL.String()
			proto  = r.Proto
			tid    = ctxstore.MustFrom[string](r.Context(), _traceIDKey)
		)

		userAttrs := slog.Group("user", "ip", ip)
		requestAttrs := slog.Group("request", "method", method, "url", url, "proto", proto, _traceIDKey.String(), tid)
		responseAttrs := slog.Group("repsonse", "status", mw.StatusCode, "size", mw.BytesCount)

		app.serverLogger().Info("access", userAttrs, requestAttrs, responseAttrs)
	})
}

func (app *application) CORS(next http.Handler) http.Handler {
	return cors.AllowAll().Handler(next)
}

func genTraceID() string {
	id, _ := uuid.NewRandom()
	return id.String()
}
