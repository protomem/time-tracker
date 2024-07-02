package model

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrNotFound = errors.New("not found")
	ErrExists   = errors.New("already exists")
)

func NewError(model string, err error) error {
	return fmt.Errorf("%s %w", capitalize(model), err)
}

func capitalize(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}
