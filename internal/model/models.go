package model

import "time"

type ID = uint

type User struct {
	ID        ID        `json:"id" db:"id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	Name       string  `json:"name" db:"name"`
	Surname    string  `json:"surname" db:"surname"`
	Patronymic *string `json:"patronymic,omitempty" db:"patronymic"`

	PassportSerie  int `json:"passportSerie" db:"passport_serie"`
	PassportNumber int `json:"passwortNumber" db:"passport_number"`

	Address string `json:"address" db:"address"`
}

type Session struct {
	ID        ID        `json:"id" db:"id"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`

	Begin time.Time  `json:"begin" db:"sess_begin"`
	End   *time.Time `json:"end" db:"sess_end"`

	Task ID `json:"taskId" db:"task_id"`
	User ID `json:"userId" db:"user_id"`
}
