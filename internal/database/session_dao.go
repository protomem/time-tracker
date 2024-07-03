package database

import (
	"context"
	"log/slog"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/protomem/time-tracker/internal/model"
)

type SessionDAO struct {
	Logger *slog.Logger
	*DB
}

func NewSessionDAO(logger *slog.Logger, db *DB) *SessionDAO {
	return &SessionDAO{
		Logger: logger.With("dao", "session"),
		DB:     db,
	}
}

func (dao *SessionDAO) LastByTaskAndUser(ctx context.Context, task, user model.ID) (model.Session, error) {
	logger := dao.Logger.With("query", "lastByTaskAndUser")

	query, args, err := dao.Builder.
		Select("*").
		From("sessions").
		Where(squirrel.Eq{"task_id": task}).
		Where(squirrel.Eq{"user_id": user}).
		OrderBy("sess_begin DESC").
		Limit(1).
		ToSql()
	if err != nil {
		return model.Session{}, err
	}

	logger.Debug("build query", "sql", query, "args", args)

	var session model.Session
	row := dao.QueryRowxContext(ctx, query, args...)
	if err := row.StructScan(&session); err != nil {
		logger.Debug("failed query execute", "error", err)

		if IsNoRows(err) {
			return model.Session{}, model.NewError("session", model.ErrNotFound)
		}

		return model.Session{}, err
	}

	logger.Debug("success query execute", "session", session)

	return session, nil
}

type InsertSessionDTO struct {
	User  model.ID
	Task  model.ID
	Begin time.Time
}

func (dao *SessionDAO) Insert(ctx context.Context, dto InsertSessionDTO) (model.ID, error) {
	logger := dao.Logger.With("query", "insert")

	query, args, err := dao.Builder.
		Insert("sessions").
		Columns("user_id", "task_id", "sess_begin").
		Values(dto.User, dto.Task, dto.Begin).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, err
	}

	logger.Debug("build query", "sql", query, "args", args)

	var id model.ID
	row := dao.QueryRowxContext(ctx, query, args...)
	if err := row.Scan(&id); err != nil {
		logger.Warn("failed query execute", "error", err)

		if IsUniqueViolation(err) {
			return 0, model.NewError("session", model.ErrExists)
		}

		return 0, err
	}

	logger.Debug("success query execute", "insertId", id)

	return id, nil
}

type UpdateSessionDTO struct {
	End time.Time
}

func (dao *SessionDAO) Update(ctx context.Context, id model.ID, dto UpdateSessionDTO) error {
	logger := dao.Logger.With("query", "update")

	query, args, err := dao.Builder.
		Update("sessions").
		SetMap(map[string]any{
			"updated_at": time.Now(),
			"sess_end":   dto.End,
		}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}

	logger.Debug("build query", "sql", query, "args", args)

	if _, err = dao.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	logger.Debug("success query execute", "updateId", id, "countUpdatedFields", 2)

	return nil
}
