package database

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/protomem/time-tracker/assets"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

const (
	_defaultTimeout = 3 * time.Second
	_driverName     = "pgx"
)

type DB struct {
	*sqlx.DB
	Builder squirrel.StatementBuilderType
	Logger  *slog.Logger
}

func New(logger *slog.Logger, dsn string, automigrate bool) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), _defaultTimeout)
	defer cancel()

	logger = logger.With("module", "database")
	dsn = dsn + "?sslmode=disable" // disable SSL

	db, err := sqlx.ConnectContext(ctx, _driverName, "postgres://"+dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxIdleTime(5 * time.Minute)
	db.SetConnMaxLifetime(2 * time.Hour)

	logger.Info("connect to database", "dsn", dsn) // TODO: hide credentials

	if automigrate {
		iofsDriver, err := iofs.New(assets.EmbeddedFiles, "migrations")
		if err != nil {
			return nil, err
		}

		migrator, err := migrate.NewWithSourceInstance("iofs", iofsDriver, "postgres://"+dsn)
		if err != nil {
			return nil, err
		}

		logger.Info("migrating database")

		err = migrator.Up()
		switch {
		case errors.Is(err, migrate.ErrNoChange):
			break
		case err != nil:
			return nil, err
		}
	}

	return &DB{
		DB:      db,
		Builder: squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		Logger:  logger,
	}, nil
}

func (db *DB) Close() error {
	db.Logger.Info("disconnect from database")
	return db.DB.Close()
}
