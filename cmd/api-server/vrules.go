package main

import (
	"github.com/protomem/time-tracker/internal/database"
	"github.com/protomem/time-tracker/internal/validator"
)

// Validation rules

func validateFindUserFilter(v *validator.Validator, filter database.FindUserFilter) {
	if filter.Name != nil {
		validateUserName(v, *filter.Name)
	}
	if filter.Surname != nil {
		validateUserSurname(v, *filter.Surname)
	}
	if filter.Patronymic != nil {
		validateUserPatronymic(v, *filter.Patronymic)
	}
	if filter.PassportSerie != nil {
		validatePassportSerie(v, *filter.PassportSerie)
	}
	if filter.PassportNumber != nil {
		validatePassportNumber(v, *filter.PassportNumber)
	}
	if filter.Address != nil {
		validateAddress(v, *filter.Address)
	}
}

func validateRequestUpdateUser(v *validator.Validator, request requestUpdateUser) {
	if request.Name != nil {
		validateUserName(v, *request.Name)
	}
	if request.Surname != nil {
		validateUserSurname(v, *request.Surname)
	}
	if request.Patronymic != nil {
		validateUserPatronymic(v, *request.Patronymic)
	}
	if request.PassportSerie != nil {
		validatePassportSerie(v, *request.PassportSerie)
	}
	if request.PassportNumber != nil {
		validatePassportNumber(v, *request.PassportNumber)
	}
	if request.Address != nil {
		validateAddress(v, *request.Address)
	}
}

func validateUserName(v *validator.Validator, userName string) {
	v.CheckField(validator.NotBlank(userName), "name", "cannot be blank")
}

func validateUserSurname(v *validator.Validator, userSurname string) {
	v.CheckField(validator.NotBlank(userSurname), "surname", "cannot be blank")
}

func validateUserPatronymic(v *validator.Validator, userPatronymic string) {
	v.CheckField(validator.NotBlank(userPatronymic), "patronymic", "cannot be blank")
}

func validatePassportSerie(v *validator.Validator, passportSerie int) {
	v.CheckField(
		passportSerie >= 0,
		"passportSerie",
		"must not negative number",
	)
}

func validatePassportNumber(v *validator.Validator, passportNumber int) {
	v.CheckField(
		passportNumber >= 0,
		"passportNumber",
		"must not negative number",
	)
}

func validateAddress(v *validator.Validator, address string) {
	v.CheckField(validator.NotBlank(address), "address", "cannot be blank")
}
