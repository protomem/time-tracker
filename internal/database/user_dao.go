package database

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/protomem/time-tracker/internal/model"
)

type UserDAO struct {
	Logger *slog.Logger
	*DB
}

func NewUserDAO(logger *slog.Logger, db *DB) *UserDAO {
	return &UserDAO{
		Logger: logger,
		DB:     db,
	}
}

func (dao *UserDAO) Get(ctx context.Context, id model.ID) (model.User, error) {
	query, args, err := dao.Builder.
		Select("*").
		From("users").
		Where(squirrel.Eq{"id": id}).
		Limit(1).
		ToSql()
	if err != nil {
		return model.User{}, err
	}

	dao.Logger.Debug("query", "sql", query, "args", args)

	var user model.User
	row := dao.QueryRowxContext(ctx, query, args...)
	if err := row.StructScan(&user); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.User{}, model.NewError("user", model.ErrNotFound)
		}

		return model.User{}, err
	}

	return user, nil
}

type InsertUserDTO struct {
	Name           string
	Surname        string
	Patronymic     *string
	PassportSerie  int
	PassportNumber int
	Address        string
}

func (dao *UserDAO) Insert(ctx context.Context, dto InsertUserDTO) (model.ID, error) {
	query, args, err := dao.Builder.
		Insert("users").
		Columns("name", "surname", "patronymic", "passport_serie", "passport_number", "address").
		Values(dto.Name, dto.Surname, dto.Patronymic, dto.PassportSerie, dto.PassportNumber, dto.Address).
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
			return 0, model.NewError("user", model.ErrExists)
		}

		return 0, err
	}

	return id, nil
}
