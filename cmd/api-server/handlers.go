package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/protomem/time-tracker/internal/ctxstore"
	"github.com/protomem/time-tracker/internal/database"
	"github.com/protomem/time-tracker/internal/external_api/people_service"
	"github.com/protomem/time-tracker/internal/model"
	"github.com/protomem/time-tracker/internal/request"
	"github.com/protomem/time-tracker/internal/response"
	"github.com/protomem/time-tracker/internal/validator"
)

// Handle Status
// @Summary Server Status
// @Description Check if the server is up and running
// @Tags api
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /status [get]
func (app *application) handleStatus(w http.ResponseWriter, r *http.Request) {
	if err := response.JSON(w, http.StatusOK, response.JSONObject{"status": "OK"}); err != nil {
		app.serverError(w, r, err)
	}
}

// Handle Add User
// @Summary Add User
// @Description Add new user
// @Tags users
// @Accept json
// @Produce json
// @Param input body main.requestAddUser true "Passport serie and number"
// @Success 201 {object} main.responseAddUser
// @Failure 400 {object} any "Bad request input"
// @Failure 422 {object} validator.Validator "Invalid input data"
// @Failure 409 {object} any "User already exists"
// @Failure 500 {object} any "Internal server error"
// @Router /users [post]
func (app *application) handleAddUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := app.logger.With(
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

	var input requestAddUser
	if err := request.DecodeJSONStrict(w, r, &input); err != nil {
		app.badRequest(w, r, err)
		return
	}

	var v validator.Validator
	v.Check(validator.NotBlank(input.PassportNumber), "Passport data cannot be blank")

	passportNumber, passportSerie, err := parsePassportNumber(input.PassportNumber)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	v.Check(validator.DigitsInNumber(passportSerie, 4), "Passport serie is not valid")
	v.Check(validator.DigitsInNumber(passportNumber, 6), "Passport number is not valid")

	if v.HasErrors() {
		app.failedValidation(w, r, v)
		return
	}

	peopleClient, err := people_service.NewClient(app.config.peopleServ.serverURL)
	if err != nil {
		app.serverError(w, r, err)
		return
	}

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

	logger.Debug("get user from people service", "user", people)

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

	dao := database.NewUserDAO(logger, app.db)

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

	if err := response.JSON(w, http.StatusCreated, responseAddUser{User: user}); err != nil {
		app.serverError(w, r, err)
		return
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

// Handle Delete User
// @Summary Delete User
// @Description Delete user
// @Tags users
// @Produce json
// @Param userID path int true "User ID"
// @Success 204
// @Failure 400 {object} any "Bad request input"
// @Failure 404 {object} any "User not found"
// @Failure 500 {object} any "Internal server error"
// @Router /users/{userId} [delete]
func (app *application) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	logger := app.logger.With(
		_traceIDKey.String(), ctxstore.MustFrom[string](ctx, _traceIDKey),
	)

	userID, err := userIDFromRequest(r)
	if err != nil {
		app.badRequest(w, r, err)
		return
	}

	dao := database.NewUserDAO(logger, app.db)

	if _, err := dao.Get(ctx, userID); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			app.errorMessage(w, r, http.StatusNotFound, err.Error(), nil)
			return
		}

		app.serverError(w, r, err)
		return
	}

	if err := dao.Delete(ctx, userID); err != nil {
		app.serverError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
