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
		Logger: logger,
		DB:     db,
	}
}

func (dao *SessionDAO) GetByTaskAndUser(ctx context.Context, task, user model.ID) (model.Session, error) {
	query, args, err := dao.Builder.
		Select("*").
		From("sessions").
		Where(squirrel.Eq{"task_id": task}).
		Where(squirrel.Eq{"user_id": user}).
		ToSql()
	if err != nil {
		return model.Session{}, err
	}

	dao.Logger.Debug("query", "sql", query, "args", args)

	var session model.Session
	row := dao.QueryRowxContext(ctx, query, args...)
	if err := row.StructScan(&session); err != nil {
		if IsNoRows(err) {
			return model.Session{}, model.NewError("session", model.ErrNotFound)
		}

		return model.Session{}, err
	}

	return session, nil
}

type InsertSessionDTO struct {
	User  model.ID
	Task  model.ID
	Begin time.Time
}

func (dao *SessionDAO) Insert(ctx context.Context, dto InsertSessionDTO) (model.ID, error) {
	query, args, err := dao.Builder.
		Insert("sessions").
		Columns("user_id", "task_id", "sess_begin").
		Values(dto.User, dto.Task, dto.Begin).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return 0, err
	}

	dao.Logger.Debug("query", "sql", query, "args", args)

	var id model.ID
	row := dao.QueryRowxContext(ctx, query, args...)
	if err := row.Scan(&id); err != nil {
		if IsUniqueViolation(err) {
			return 0, model.NewError("session", model.ErrExists)
		}

		return 0, err
	}

	return id, nil
}

type UpdateSessionDTO struct {
	End time.Time
}

func (dao *SessionDAO) Update(ctx context.Context, id model.ID, dto UpdateSessionDTO) error {
	query, args, err := dao.Builder.
		Update("sessions").
		SetMap(map[string]any{
			"sess_end": dto.End,
		}).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}

	dao.Logger.Debug("query", "sql", query, "args", args)

	if _, err = dao.ExecContext(ctx, query, args...); err != nil {
		return err
	}

	return nil
}
