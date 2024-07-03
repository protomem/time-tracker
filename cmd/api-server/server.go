package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	_defaultIdleTimeout    = time.Minute
	_defaultReadTimeout    = 5 * time.Second
	_defaultWriteTimeout   = 10 * time.Second
	_defaultShutdownPeriod = 30 * time.Second
)

func (app *application) serveHTTP() error {
	app.confiureSwagger()

	srv := &http.Server{
		Addr:         fmtHTTPAddr(app.config.httpHost, app.config.httpPort),
		Handler:      app.routes(),
		ErrorLog:     slog.NewLogLogger(app.baseLogger.Handler(), slog.LevelWarn),
		IdleTimeout:  _defaultIdleTimeout,
		ReadTimeout:  _defaultReadTimeout,
		WriteTimeout: _defaultWriteTimeout,
	}

	shutdownErrorChan := make(chan error)

	go func() {
		quitChan := make(chan os.Signal, 1)
		signal.Notify(quitChan, syscall.SIGINT, syscall.SIGTERM)
		<-quitChan

		ctx, cancel := context.WithTimeout(context.Background(), _defaultShutdownPeriod)
		defer cancel()

		shutdownErrorChan <- srv.Shutdown(ctx)
	}()

	app.serverLogger().Info("starting server", slog.Group("server", "addr", srv.Addr))

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownErrorChan
	if err != nil {
		return err
	}

	app.serverLogger().Info("stopped server", slog.Group("server", "addr", srv.Addr))

	app.wg.Wait()
	return nil
}

func (app *application) serverLogger(args ...any) *slog.Logger {
	args = append(args, "module", "server")
	return app.baseLogger.With(args...)
}

func fmtHTTPAddr(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}
