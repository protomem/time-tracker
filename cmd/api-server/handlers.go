package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/protomem/time-tracker/internal/database"
	"github.com/protomem/time-tracker/internal/external_api/people_service"
	"github.com/protomem/time-tracker/internal/model"
	"github.com/protomem/time-tracker/internal/request"
	"github.com/protomem/time-tracker/internal/response"
	"github.com/protomem/time-tracker/internal/validator"
	"github.com/samber/lo"
)

// Handle Status
//
//	@Summary		Server Status
//	@Description	Check if the server is up and running
//	@Tags			api
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string
//	@Router			/status [get]
func (app *application) handleStatus(w http.ResponseWriter, r *http.Request) {
	if err := response.JSON(w, http.StatusOK, response.JSONObject{"status": "OK"}); err != nil {
		app.serverError(w, r, err)
	}
}

// Handle Find Users
//
//	@Summary		Find Users
//	@Description	Get all users by filters with pagination
//	@Tags			users
//	@Produce		json
//	@Param			page			query		int		false	"Page number"	default(1)	minimum(1)
//	@Param			pageSize		query		int		false	"Page size"		default(10)	minimum(1)
//	@Param			name			query		string	false	"User name"
//	@Param			surname			query		string	false	"User surname"
//	@Param			patronymic		query		string	false	"User patronymic"
//	@Param			address			query		string	false	"User address"
//	@Param			passportSerie	query		int		false	"User passport serie"
//	@Param			passportNumber	query		int		false	"User passport number"
//	@Success		200				{array}		model.User
//	@Failure		400				{object}	any					"Bad request"
//	@Failure		422				{object}	validator.Validator	"Invalid input data"
//	@Failure		500				{object}	any					"Internal server error"
//	@Router			/users [get]
func (app *application) handleFindUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "findUsers")

	opts := findOptionsFromRequest(r)
	filter := findUserFilterFromRequest(r)

	if v := validator.Validate(func(v *validator.Validator) {
		validateFindUserFilter(v, filter)
	}); v.HasErrors() {
		app.failedValidation(w, r, v)
		return
	}

	handlerLogger.Debug("read params and body", "filter", filter, "opts", opts)

	users, err := findUsers(ctx, app.db, baseLogger, filter, opts)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("users found", "count", len(users))

	if err := response.JSON(w, http.StatusOK, users); err != nil {
		app.serverError(w, r, err)
	}
}

func findUsers(
	ctx context.Context, db *database.DB, logger *slog.Logger,
	filter database.FindUserFilter, opts database.FindOptions,
) ([]model.User, error) {
	dao := database.NewUserDAO(logger, db)

	users, err := dao.Find(ctx, filter, opts)
	if err != nil {
		return []model.User{}, err
	}

	return users, nil
}

// Handle Add User
//
//	@Summary		Add User
//	@Description	Add new user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			input	body		main.requestAddUser	true	"Passport serie and number"
//	@Success		201		{object}	model.User
//	@Failure		400		{object}	any					"Bad request input"
//	@Failure		409		{object}	any					"User already exists"
//	@Failure		422		{object}	validator.Validator	"Invalid input data"
//	@Failure		500		{object}	any					"Internal server error"
//	@Router			/users [post]
func (app *application) handleAddUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "addUser")

	var input requestAddUser
	if err := request.DecodeJSONStrict(w, r, &input); err != nil {
		app.badRequest(w, r, err)
		return
	}

	var (
		err                           error
		passportSerie, passportNumber int
	)

	if v := validator.Validate(func(v *validator.Validator) {
		passportSerie, passportNumber, err = parsePassportNumber(input.PassportNumber)
		if err != nil {
			v.AddFieldError("passportNumber", err.Error())
			return
		}

		validatePassportSerie(v, passportSerie)
		validatePassportNumber(v, passportNumber)
	}); v.HasErrors() {
		app.failedValidation(w, r, v)
		return
	}

	handlerLogger.Debug("read params and body", "passportSerie", passportSerie, "passportNumber", passportNumber)

	people, err := fetchPeople(ctx, baseLogger, app.config.peopleServ.serverURL, passportSerie, passportNumber)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	// TODO: Validate people ?

	user, err := insertUser(ctx, app.db, baseLogger, people, passportSerie, passportNumber)
	if err != nil {
		if errors.Is(err, model.ErrExists) {
			app.errorMessage(w, r, http.StatusConflict, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("inserted user", "userId", user.ID)

	if err := response.JSON(w, http.StatusCreated, user); err != nil {
		app.serverError(w, r, err)
	}
}

type requestAddUser struct {
	PassportNumber string `json:"passportNumber"`
}

func parsePassportNumber(s string) (passportSerie int, passportNumber int, err error) {
	parts := strings.Split(s, " ")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid passport data")
	}

	if passportSerie, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, fmt.Errorf("invalid passport serie: not a number")
	}
	if passportNumber, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, fmt.Errorf("invalid passport number: not a number")
	}

	return
}

func fetchPeople(
	ctx context.Context, logger *slog.Logger, addr string,
	passportSerie int, passportNumber int,
) (*people_service.People, error) {
	logger.Debug("connect to people service", "addr", addr)

	client, err := people_service.NewClient(addr)
	if err != nil {
		return nil, err
	}

	logger.Debug("do people request", "passportSerie", passportSerie, "passportNumber", passportNumber)

	infoPeopleReq, err := client.InfoGet(ctx, people_service.InfoGetParams{
		PassportSerie:  passportSerie,
		PassportNumber: passportNumber,
	})
	if err != nil {
		return nil, err
	}

	people, ok := infoPeopleReq.(*people_service.People)
	if !ok {
		return nil, fmt.Errorf("invalid response from people service: %T", infoPeopleReq)
	}

	logger.Debug("fetch people", "people", people)

	return people, nil
}

func insertUser(
	ctx context.Context, db *database.DB, logger *slog.Logger,
	people *people_service.People, passportSerie int, passportNumber int,
) (model.User, error) {
	dao := database.NewUserDAO(logger, db)

	insertDTO := database.NewInsertUserDTO(
		people.GetName(), people.GetSurname(),
		passportSerie, passportNumber,
		people.GetAddress(),
	)
	if people.GetPatronymic().Set {
		insertDTO.SetPatronymic(people.GetPatronymic().Value)
	}

	userID, err := dao.Insert(ctx, insertDTO)
	if err != nil {
		if errors.Is(err, model.ErrExists) {
			return model.User{}, model.NewError("user", model.ErrExists)
		}

		return model.User{}, err
	}

	user, err := dao.Get(ctx, userID)
	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

// Handle Update User
//
//	@Summary		Update user
//	@Description	Update user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			userId	path		int						true	"User ID"
//	@Param			input	body		main.requestUpdateUser	true	"New user data"
//	@Success		200		{object}	model.User
//	@Failure		400		{object}	any					"Bad request"
//	@Failure		404		{object}	any					"User not found"
//	@Failure		409		{object}	any					"User already exists"
//	@Failure		422		{object}	validator.Validator	"Invalid input data"
//	@Failure		500		{object}	any					"Internal server error"
//	@Router			/users/{userId} [put]
func (app *application) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "updateUser")

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	var input requestUpdateUser
	err = request.DecodeJSON(w, r, &input)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	if v := validator.Validate(func(v *validator.Validator) {
		validateRequestUpdateUser(v, input)
	}); v.HasErrors() {
		app.failedValidation(w, r, v)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID, "input", input)

	user, err := updateUser(ctx, app.db, baseLogger, userID, input)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("user updated", "updatedUserId", user.ID)

	if err := response.JSON(w, http.StatusOK, user); err != nil {
		app.serverError(w, r, err)
	}
}

type requestUpdateUser struct {
	Name           *string `json:"name"`
	Surname        *string `json:"surname"`
	Patronymic     *string `json:"patronymic"`
	PassportSerie  *int    `json:"passportSerie"`
	PassportNumber *int    `json:"passportNumber"`
	Address        *string `json:"address"`
}

func updateUser(
	ctx context.Context, db *database.DB, logger *slog.Logger,
	userID model.ID, requestBody requestUpdateUser,
) (model.User, error) {
	dao := database.NewUserDAO(logger, db)

	logger.Debug("check exists user", "userId", userID)

	if _, err := dao.Get(ctx, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.User{}, model.NewError("user", model.ErrNotFound)
		}

		return model.User{}, err
	}

	dto := database.UpdateUserDTO(requestBody)

	if err := dao.Update(ctx, userID, dto); err != nil {
		return model.User{}, err
	}

	user, err := dao.Get(ctx, userID)
	if err != nil {
		return model.User{}, err
	}

	logger.Debug("update user", "userId", userID)

	return user, nil
}

// Handle Delete User
//
//	@Summary		Delete User
//	@Description	Delete user
//	@Tags			users
//	@Produce		json
//	@Param			userId	path	int	true	"User ID"
//	@Success		204
//	@Failure		400	{object}	any	"Bad request input"
//	@Failure		404	{object}	any	"User not found"
//	@Failure		500	{object}	any	"Internal server error"
//	@Router			/users/{userId} [delete]
func (app *application) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "deleteUser")

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID)

	if err := deleteUser(ctx, app.db, baseLogger, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func deleteUser(
	ctx context.Context, db *database.DB, logger *slog.Logger,
	userID model.ID,
) error {
	dao := database.NewUserDAO(logger, db)

	logger.Debug("check exists user", "userId", userID)

	if _, err := dao.Get(ctx, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewError("user", model.ErrNotFound)
		}

		return err
	}

	logger.Debug("delete user", "userId", userID)

	if err := dao.Delete(ctx, userID); err != nil {
		return err
	}

	return nil
}

// Handle Find Sessions
//
//	@Summary		Find Sessions
//	@Description	Get all user sessions
//	@Tags			sessions
//	@Produce		json
//	@Param			userId	path		int	true	"User ID"
//	@Success		200		{object}	[]model.Session
//	@Failure		400		{object}	any	"Bad request input"
//	@Failure		404		{object}	any	"User not found"
//	@Failure		500		{object}	any	"Internal server error"
//	@Router			/sessions/{userId} [get]
func (app *application) handleFindSessions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "findSessions")

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID)

	if err := checkUserExists(ctx, app.db, baseLogger, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	// TODO: Sort sessions

	sessions, err := findSessions(ctx, app.db, baseLogger, userID, database.SessionTimelineOptions{})
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	if err := response.JSON(w, http.StatusOK, sessions); err != nil {
		app.serverError(w, r, err)
	}
}

// Handle Session Start
//
//	@Summary		Start Session
//	@Description	Start new session
//	@Tags			sessions
//	@Produce		json
//	@Param			userId	path	int	true	"User ID"
//	@Param			taskId	path	int	true	"Task ID"
//	@Success		201
//	@Failure		400	{object}	any	"Bad request input"
//	@Failure		409	{object}	any	"Session already exists"
//	@Failure		500	{object}	any	"Internal server error"
//	@Router			/sessions/{userId}/{taskId} [post]
func (app *application) handleSessionStart(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "sessionStart")

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	taskID, err := taskIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID, "taskId", taskID)

	if _, err := insertSessionWithCheckExistsNotEnded(ctx, app.db, baseLogger, userID, taskID); err != nil {
		if errors.Is(err, model.ErrExists) {
			app.errorMessage(w, r, http.StatusConflict, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func insertSessionWithCheckExistsNotEnded(
	ctx context.Context, db *database.DB, logger *slog.Logger,
	userID model.ID, taskID model.ID,
) (model.Session, error) {
	dao := database.NewSessionDAO(logger, db)

	logger.Debug("check exists and not ended session", "userId", userID, "taskId", taskID)

	session, err := dao.LastByTaskAndUser(ctx, taskID, userID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return model.Session{}, err
	}

	if !errors.Is(err, model.ErrNotFound) && session.End == nil {
		return model.Session{}, model.NewError("session", model.ErrExists)
	}

	logger.Debug("insert session", "userId", userID, "taskId", taskID)

	dto := database.NewInsertSessionDTO(userID, taskID)

	sessionID, err := dao.Insert(ctx, dto)
	if err != nil {
		if errors.Is(err, model.ErrExists) {
			return model.Session{}, model.NewError("session", model.ErrExists)
		}

		return model.Session{}, err
	}

	logger.Debug("get session", "sessionId", sessionID)

	session, err = dao.Get(ctx, sessionID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.Session{}, model.NewError("session", model.ErrNotFound)
		}

		return model.Session{}, err
	}

	return session, nil
}

// Handle Session Stop
//
//	@Summary		Stop Session
//	@Description	Stop session
//	@Tags			sessions
//	@Produce		json
//	@Param			userId	path	int	true	"User ID"
//	@Param			taskId	path	int	true	"Task ID"
//	@Success		204
//	@Failure		400	{object}	any	"Bad request input"
//	@Failure		404	{object}	any	"Session not found"
//	@Failure		500	{object}	any	"Internal server error"
//	@Router			/sessions/{userId}/{taskId} [delete]
func (app *application) handleSessionStop(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "sessionStop")

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	taskID, err := taskIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID, "taskId", taskID)

	if _, err := updateSessionEnd(ctx, app.db, baseLogger, userID, taskID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateSessionEnd(
	ctx context.Context, db *database.DB, logger *slog.Logger,
	userID model.ID, taskID model.ID,
) (model.Session, error) {
	dao := database.NewSessionDAO(logger, db)

	logger.Debug("check exists and not ended session", "userId", userID, "taskId", taskID)

	session, err := dao.LastByTaskAndUser(ctx, taskID, userID)
	if err != nil || session.End != nil {
		if errors.Is(err, model.ErrNotFound) || session.End != nil {
			return model.Session{}, model.NewError("session", model.ErrNotFound)
		}

		return model.Session{}, err
	}

	logger.Debug("update session", "userId", userID, "taskId", taskID)

	if err := dao.Update(ctx, session.ID, database.UpdateSessionDTO{
		End: time.Now(),
	}); err != nil {
		return model.Session{}, err
	}

	logger.Debug("get session", "sessionId", session.ID)

	session, err = dao.Get(ctx, session.ID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.Session{}, model.NewError("session", model.ErrNotFound)
		}

		return model.Session{}, err
	}

	return session, nil
}

// Handle User Stats
//
//	@Summary		Users Statistics
//	@Description	Get users statistics
//	@Tags			users
//	@Produce		json
//	@Param			userId	path		int		true	"User ID"
//	@Param			after	query		string	false	"Start date"
//	@Param			before	query		string	false	"End date"
//	@Success		200		{array}		main.userFormatStat
//	@Failure		400		{object}	any	"Bad request input"
//	@Failure		404		{object}	any	"User not found"
//	@Failure		500		{object}	any	"Internal server error"
//	@Router			/users/{userId}/stats [get]
func (app *application) handleUserStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger, handlerLogger := app.buildHandlerLoggers(r, "userStats")

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	opts, err := sessionTimelineOptionsFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID, "opts", opts)

	if err := checkUserExists(ctx, app.db, baseLogger, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		return
	}

	sessions, err := findSessions(ctx, app.db, baseLogger, userID, opts)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	stats := mapSessionsToUserFormatStats(sessions, opts)

	if err := response.JSON(w, http.StatusOK, stats); err != nil {
		app.serverError(w, r, err)
	}
}

type userStat struct {
	Task model.ID      `json:"task"`
	Time time.Duration `json:"time"`
}

type userFormatStat struct {
	Task model.ID `json:"task"`
	Time string   `json:"time"`
}

func newUserFormatStat(s userStat) userFormatStat {
	return userFormatStat{
		Task: s.Task,
		Time: s.Time.String(), // TODO: Pretty format
	}
}

func checkUserExists(ctx context.Context, db *database.DB, logger *slog.Logger, userID model.ID) error {
	dao := database.NewUserDAO(logger, db)

	logger.Debug("check user exists", "userId", userID)

	if _, err := dao.Get(ctx, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return model.NewError("user", model.ErrNotFound)
		}

		return err
	}

	return nil
}

func findSessions(
	ctx context.Context, db *database.DB, logger *slog.Logger,
	userID model.ID, opts database.SessionTimelineOptions,
) ([]model.Session, error) {
	dao := database.NewSessionDAO(logger, db)

	logger.Debug("find sessions", "userId", userID, "opts", opts)

	sessions, err := dao.FindByUser(ctx, userID, opts)
	if err != nil {
		return []model.Session{}, err
	}

	return sessions, nil
}

func mapSessionsToUserFormatStats(sessions []model.Session, opts database.SessionTimelineOptions) []userFormatStat {
	grouped := lo.GroupBy(sessions, func(session model.Session) model.ID {
		return session.Task
	})

	stats := lo.MapToSlice(grouped, func(task model.ID, sessions []model.Session) userStat {
		return userStat{
			Task: task,
			Time: calcSumSessions(sessions, opts),
		}
	})

	slices.SortFunc(stats, func(a, b userStat) int {
		return cmp.Compare(a.Time, b.Time)
	})

	return lo.Map(stats, func(session userStat, _ int) userFormatStat {
		return newUserFormatStat(session)
	})
}

func calcSumSessions(sessions []model.Session, opts database.SessionTimelineOptions) time.Duration {
	return lo.SumBy(sessions, func(session model.Session) time.Duration {
		if opts.After != nil && session.Begin.After(*opts.After) {
			session.Begin = *opts.After
		}

		if session.End == nil || (opts.Before != nil && session.End.After(*opts.Before)) {
			session.End = new(time.Time)
			if opts.Before != nil {
				*session.End = *opts.Before
			} else {
				*session.End = time.Now()
			}
		}

		return session.End.Sub(session.Begin)
	})
}
