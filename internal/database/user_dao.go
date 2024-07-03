package database

import (
	"context"
	"log/slog"
	"time"

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

type FindUserFilter struct {
	Name           *string
	Surname        *string
	Patronymic     *string
	PassportSerie  *int
	PassportNumber *int
	Address        *string
}

func (dao *UserDAO) Find(ctx context.Context, filter FindUserFilter, opts FindOptions) ([]model.User, error) {
	equals := squirrel.Eq{}
	if filter.Name != nil {
		equals["name"] = *filter.Name
	}
	if filter.Surname != nil {
		equals["surname"] = *filter.Surname
	}
	if filter.Patronymic != nil {
		equals["patronymic"] = *filter.Patronymic
	}
	if filter.PassportSerie != nil {
		equals["passport_serie"] = *filter.PassportSerie
	}
	if filter.PassportNumber != nil {
		equals["passport_number"] = *filter.PassportNumber
	}
	if filter.Address != nil {
		equals["address"] = *filter.Address
	}

	query, args, err := dao.Builder.
		Select("*").
		From("users").
		Where(equals).
		Limit(uint64(opts.Limit)).
		Offset(uint64(opts.Offset)).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return []model.User{}, err
	}

	dao.Logger.Debug("query", "sql", query, "args", args)

	users := make([]model.User, 0, opts.Limit)
	if err := dao.SelectContext(ctx, &users, query, args...); err != nil {
		if IsNoRows(err) {
			return []model.User{}, nil
		}

		return []model.User{}, err
	}

	return users, nil
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
		if IsNoRows(err) {
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

type UpdateUserDTO struct {
	Name           *string
	Surname        *string
	Patronymic     *string
	PassportSerie  *int
	PassportNumber *int
	Address        *string
}

func (dao *UserDAO) Update(ctx context.Context, id model.ID, dto UpdateUserDTO) error {
	data := make(map[string]any, 7)
	data["updated_at"] = time.Now()
	if dto.Name != nil {
		data["name"] = *dto.Name
	}
	if dto.Surname != nil {
		data["surname"] = *dto.Surname
	}
	if dto.Patronymic != nil {
		data["patronymic"] = *dto.Patronymic
	}
	if dto.PassportSerie != nil {
		data["passport_serie"] = *dto.PassportSerie
	}
	if dto.PassportNumber != nil {
		data["passport_number"] = *dto.PassportNumber
	}
	if dto.Address != nil {
		data["address"] = *dto.Address
	}

	query, args, err := dao.Builder.
		Update("users").
		SetMap(data).
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return err
	}

	dao.Logger.Debug("query", "sql", query, "args", args)

	if _, err = dao.ExecContext(ctx, query, args...); err != nil {
		if IsUniqueViolation(err) {
			return model.NewError("user", model.ErrExists)
		}

		return err
	}

	return nil
}

func (dao *UserDAO) Delete(ctx context.Context, id model.ID) error {
	query, args, err := dao.Builder.
		Delete("users").
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
