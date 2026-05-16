package entity

import "errors"

var (
	ErrNationalIDAlreadyExists  = errors.New("national ID already exists")
	ErrPhoneNumberAlreadyExists = errors.New("phone number already exists")
	ErrEmailAlreadyExists      = errors.New("email already exists")
)
