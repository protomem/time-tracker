package main

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/protomem/time-tracker/internal/ctxstore"
	"github.com/protomem/time-tracker/internal/database"
	"github.com/protomem/time-tracker/internal/external_api/people_service"
	"github.com/protomem/time-tracker/internal/model"
	"github.com/protomem/time-tracker/internal/request"
	"github.com/protomem/time-tracker/internal/response"
	"github.com/protomem/time-tracker/internal/validator"
	"github.com/samber/lo"
)

const (
	_defaultPage     = 1
	_defaultPageSize = 10
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

// Handle Show Users
//
//	@Summary		Show Users
//	@Description	Show all users by filters with pagination
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
//	@Success		200				{object}	main.responseShowUsers
//	@Failure		400				{object}	any					"Bad request"
//	@Failure		422				{object}	validator.Validator	"Invalid input data"
//	@Failure		500				{object}	any					"Internal server error"
//	@Router			/users [get]
func (app *application) handleShowUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger := app.baseLogger.With(_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey))
	handlerLogger := app.serverLogger(
		"handler", "showUsers",
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

	// TODO: Add validation

	page := defaultIntQueryParams(r, "page", _defaultPage)
	pageSize := defaultIntQueryParams(r, "pageSize", _defaultPageSize)

	opts := database.FindOptions{
		Limit:  pageSize,
		Offset: pageSize * (page - 1),
	}

	filter := database.FindUserFilter{
		Name:           optionalStringQueryParams(r, "name"),
		Surname:        optionalStringQueryParams(r, "surname"),
		Patronymic:     optionalStringQueryParams(r, "patronymic"),
		PassportSerie:  optionalIntQueryParams(r, "passportSerie"),
		PassportNumber: optionalIntQueryParams(r, "passportNumber"),
		Address:        optionalStringQueryParams(r, "address"),
	}

	handlerLogger.Debug("read params and body", "filter", filter, "opts", opts)

	dao := database.NewUserDAO(baseLogger, app.db)
	users, err := dao.Find(ctx, filter, opts)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("users found", "count", len(users))

	if err := response.JSON(w, http.StatusOK, responseShowUsers{Users: users}); err != nil {
		app.serverError(w, r, err)
	}
}

type responseShowUsers struct {
	Users []model.User `json:"users"`
}

// Handle Add User
//
//	@Summary		Add User
//	@Description	Add new user
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			input	body		main.requestAddUser	true	"Passport serie and number"
//	@Success		201		{object}	main.responseAddUser
//	@Failure		400		{object}	any					"Bad request input"
//	@Failure		409		{object}	any					"User already exists"
//	@Failure		422		{object}	validator.Validator	"Invalid input data"
//	@Failure		500		{object}	any					"Internal server error"
//	@Router			/users [post]
func (app *application) handleAddUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger := app.baseLogger.With(_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey))
	handlerLogger := app.serverLogger().With(
		"handler", "addUser",
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

	var input requestAddUser
	if err := request.DecodeJSONStrict(w, r, &input); err != nil {
		app.badRequest(w, r, err)
		return
	}

	var v validator.Validator
	v.CheckField(validator.NotBlank(input.PassportNumber), "passportNumber", "cannot be blank")

	passportNumber, passportSerie, err := parsePassportNumber(input.PassportNumber)
	if err != nil {
		// TODO: bad request -> failed validation
		app.badRequest(w, r, err)
		return
	}

	v.Check(validator.DigitsInNumber(passportSerie, 4), "Passport serie is not valid")
	v.Check(validator.DigitsInNumber(passportNumber, 6), "Passport number is not valid")

	if v.HasErrors() {
		app.failedValidation(w, r, v)
		return
	}

	handlerLogger.Debug("read params and body", "passportNumber", passportNumber, "passportSerie", passportSerie)

	peopleClient, err := people_service.NewClient(app.config.peopleServ.serverURL)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("create connection to people service", "addr", app.config.peopleServ.serverURL)

	infoPeopleReq, err := peopleClient.InfoGet(ctx, people_service.InfoGetParams{
		PassportSerie:  passportSerie,
		PassportNumber: passportNumber,
	})
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	people, ok := infoPeopleReq.(*people_service.People)
	if !ok {
		// TODO: Handle errors
		app.serverError(w, r, errors.New("invalid response from people service"))
		return
	}

	handlerLogger.Debug("get user from people service", "user", people)

	insertDTO := database.InsertUserDTO{
		Name:           people.GetName(),
		Surname:        people.GetSurname(),
		PassportNumber: passportNumber,
		PassportSerie:  passportSerie,
		Address:        people.GetAddress(),
	}

	if people.GetPatronymic().Set {
		insertDTO.Patronymic = new(string)
		*insertDTO.Patronymic = people.GetPatronymic().Value
	}

	dao := database.NewUserDAO(baseLogger, app.db)

	userID, err := dao.Insert(ctx, insertDTO)
	if err != nil {
		if errors.Is(err, model.ErrExists) {
			app.errorMessage(w, r, http.StatusConflict, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	user, err := dao.Get(ctx, userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("user inserted", "user", user)

	if err := response.JSON(w, http.StatusCreated, responseAddUser{User: user}); err != nil {
		app.serverError(w, r, err)
	}
}

type requestAddUser struct {
	PassportNumber string `json:"passportNumber"`
}

type responseAddUser struct {
	User model.User `json:"user"`
}

func parsePassportNumber(s string) (passportNumber int, passportSerie int, err error) {
	parts := strings.Split(s, " ")
	if len(parts) != 2 {
		return 0, 0, errors.New("invalid passport number")
	}

	if passportSerie, err = strconv.Atoi(parts[0]); err != nil {
		return 0, 0, fmt.Errorf("invalid passport serie: %w", err)
	}
	if passportNumber, err = strconv.Atoi(parts[1]); err != nil {
		return 0, 0, fmt.Errorf("invalid passport number: %w", err)
	}

	return
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
//	@Success		200		{object}	main.responseUpdateUser
//	@Failure		400		{object}	any					"Bad request"
//	@Failure		404		{object}	any					"User not found"
//	@Failure		409		{object}	any					"User already exists"
//	@Failure		422		{object}	validator.Validator	"Invalid input data"
//	@Failure		500		{object}	any					"Internal server error"
//	@Router			/users/{userId} [put]
func (app *application) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	baseLogger := app.baseLogger.With(_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey))
	handlerLogger := app.serverLogger(
		"handler", "updateUser",
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

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

	var v validator.Validator
	// TODO: Add more validation rules

	// TODO: Add better error messages
	if input.PassportSerie != nil {
		v.CheckField(validator.DigitsInNumber(*input.PassportSerie, 4), "passportSerie", "is not valid")
	}
	if input.PassportNumber != nil {
		v.CheckField(validator.DigitsInNumber(*input.PassportNumber, 6), "passportNumber", "is not valid")
	}

	if v.HasErrors() {
		app.failedValidation(w, r, v)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID, "input", input)

	dao := database.NewUserDAO(baseLogger, app.db)

	if _, err := dao.Get(ctx, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("check if user exists", "userId", userID)

	updateDTO := database.UpdateUserDTO(input)

	if err := dao.Update(ctx, userID, updateDTO); err != nil {
		app.serverError(w, r, err)
		return
	}

	user, err := dao.Get(ctx, userID)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("user updated", "newUser", user)

	if err := response.JSON(w, http.StatusOK, responseUpdateUser{User: user}); err != nil {
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

type responseUpdateUser struct {
	User model.User `json:"user"`
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
	baseLogger := app.baseLogger.With(_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey))
	handlerLogger := app.serverLogger().With(
		"handler", "updateUser",
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	handlerLogger.Debug("read params and body", "userId", userID)

	dao := database.NewUserDAO(baseLogger, app.db)

	if _, err := dao.Get(ctx, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("check if user exists", "userId", userID)

	if err := dao.Delete(ctx, userID); err != nil {
		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("user deleted", "userId", userID)

	w.WriteHeader(http.StatusNoContent)
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
	baseLogger := app.baseLogger.With(_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey))
	handlerLogger := app.serverLogger().With(
		"handler", "updateUser",
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

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

	dao := database.NewSessionDAO(baseLogger, app.db)

	session, err := dao.LastByTaskAndUser(ctx, taskID, userID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		app.serverError(w, r, err)
		return
	}

	if !errors.Is(err, model.ErrNotFound) && session.End == nil {
		app.errorMessage(w, r, http.StatusConflict, model.NewError("session", model.ErrExists).Error(), nil)
		return
	}

	handlerLogger.Debug("check if session exists and not ended", "sessionId", session.ID)

	insertDTO := database.InsertSessionDTO{
		User:  userID,
		Task:  taskID,
		Begin: time.Now(),
	}

	if _, err := dao.Insert(ctx, insertDTO); err != nil {
		if errors.Is(err, model.ErrExists) {
			app.errorMessage(w, r, http.StatusConflict, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("session inserted", "sessionId", session.ID)

	w.WriteHeader(http.StatusCreated)
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
	baseLogger := app.baseLogger.With(_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey))
	handlerLogger := app.serverLogger(
		"handler", "updateUser",
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

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

	dao := database.NewSessionDAO(baseLogger, app.db)

	session, err := dao.LastByTaskAndUser(ctx, taskID, userID)
	if err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	if session.End != nil {
		app.errorMessage(w, r, http.StatusNotFound, model.NewError("session", model.ErrNotFound).Error(), nil)
		return
	}

	handlerLogger.Debug("check if session exists and not ended", "sessionId", session.ID)

	if err := dao.Update(ctx, session.ID, database.UpdateSessionDTO{
		End: time.Now(),
	}); err != nil {
		app.serverError(w, r, err)
		return
	}

	handlerLogger.Debug("session updated", "sessionId", session.ID)

	w.WriteHeader(http.StatusNoContent)
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
	baseLogger := app.baseLogger.With(_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey))
	handlerLogger := app.serverLogger(
		"handler", "updateUser",
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	opts := database.SessionTimelineOptions{}

	after, ok, err := timeQueryParams(r, "after")
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	if ok {
		opts.After = &after
	}

	before, ok, err := timeQueryParams(r, "before")
	if err != nil {
		app.badRequest(w, r, err)
		return
	}
	if ok {
		now := time.Now()
		if now.Before(before) {
			before = now
		}

		opts.Before = &before
	}

	handlerLogger.Debug("read params and body", "userId", userID, "opts", opts)

	userDAO := database.NewUserDAO(baseLogger, app.db)
	if _, err := userDAO.Get(ctx, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	sessDAO := database.NewSessionDAO(baseLogger, app.db)
	sessions, err := sessDAO.FindByUser(ctx, userID, opts)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

	groupedSessions := lo.GroupBy(sessions, func(session model.Session) model.ID {
		return session.Task
	})

	userStats := lo.MapToSlice(groupedSessions, func(task model.ID, sessions []model.Session) userStat {
		sum := lo.SumBy(sessions, func(session model.Session) time.Duration {
			if opts.After != nil && session.Begin.After(*opts.After) {
				session.Begin = *opts.After
			}
			if session.End == nil || session.End.After(*opts.Before) {
				session.End = new(time.Time)
				if opts.Before != nil {
					*session.End = *opts.Before
				} else {
					*session.End = time.Now()
				}
			}
			return session.End.Sub(session.Begin)
		})
		return userStat{
			Task: task,
			Time: sum,
		}
	})

	slices.SortFunc(userStats, func(a, b userStat) int {
		return int(b.Time - a.Time) // TODO: Change to duration comparison
	})

	userFormatStats := lo.Map(userStats, func(session userStat, _ int) userFormatStat {
		return newUserFormatStat(session)
	})

	if err := response.JSON(w, http.StatusOK, userFormatStats); err != nil {
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
