package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"
	"sync"

	"github.com/protomem/time-tracker/internal/database"
	"github.com/protomem/time-tracker/internal/env"
	"github.com/protomem/time-tracker/internal/version"
)

var _cfgFile = flag.String("cfg", "", "path to config file")

func init() {
	flag.Parse()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	err := run(logger)
	if err != nil {
		trace := string(debug.Stack())
		logger.Error(err.Error(), "trace", trace)
		os.Exit(1)
	}
}

type config struct {
	httpHost string
	httpPort int
	db       struct {
		dsn         string
		automigrate bool
	}
	peopleServ struct {
		serverURL string
	}
}

type application struct {
	config config
	db     *database.DB
	logger *slog.Logger
	wg     sync.WaitGroup
}

func run(logger *slog.Logger) error {
	var cfg config

	if *_cfgFile != "" {
		err := env.Load(*_cfgFile)
		if err != nil {
			return err
		}
	}

	cfg.httpHost = env.GetString("HTTP_HOST", "localhost")
	cfg.httpPort = env.GetInt("HTTP_PORT", 8080)
	cfg.db.dsn = env.GetString("DB_DSN", "postgres:postgres@localhost:5432/postgres")
	cfg.db.automigrate = env.GetBool("DB_AUTOMIGRATE", true)
	cfg.peopleServ.serverURL = env.GetString("PEOPLE_SERVICE_URL", "http://localhost:8081")

	showVersion := flag.Bool("version", false, "display version and exit")

	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\n", version.Get())
		return nil
	}

	db, err := database.New(logger, cfg.db.dsn, cfg.db.automigrate)
	if err != nil {
		return err
	}
	defer db.Close()

	app := &application{
		config: cfg,
		db:     db,
		logger: logger,
	}

	return app.serveHTTP()
}
