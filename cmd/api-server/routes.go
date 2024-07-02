package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/protomem/time-tracker/docs"

	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (app *application) confiureSwagger() {
	docs.SwaggerInfo.Title = "Time Tracker"
	docs.SwaggerInfo.Description = "Web API - Time Tracker"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmtHTTPAddr("localhost", app.config.httpPort)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http"}
}

func (app *application) routes() http.Handler {
	mux := chi.NewRouter()

	mux.NotFound(app.notFound)
	mux.MethodNotAllowed(app.methodNotAllowed)

	mux.Use(app.traceID)
	mux.Use(app.logAccess)
	mux.Use(app.recoverPanic)

	mux.Use(app.CORS)

	mux.Get("/api/v1/status", app.handleStatus)

	mux.Post("/api/v1/users", app.handleAddUser)
	mux.Delete("/api/v1/users/{userId}", app.handleDeleteUser)

	mux.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(
			"http://"+fmtHTTPAddr("localhost", app.config.httpPort)+"/swagger/doc.json",
		), // The url pointing to API definition
	))

	app.logger.Debug("routes configured", "routes", chiRoutesToStrings(mux.Routes()))

	return mux
}

func chiRoutesToStrings(routes []chi.Route) []string {
	parsedRoutes := make([]string, 0, len(routes))
	for _, route := range routes {
		parsedRoutes = append(parsedRoutes, route.Pattern)
	}
	return parsedRoutes
}
