package auth

import "errors"

var ErrTOTPRequired = errors.New("totp required")
