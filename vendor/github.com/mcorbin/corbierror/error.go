package corbierror

import (
	"fmt"
	"strings"
)

type ErrorType int

const (
	BadRequest ErrorType = iota + 1
	Unauthorized
	Forbidden
	NotFound
	Conflict
	Internal
)

type Error struct {
	Messages  []string  `json:"messages"`
	Cause     error     `json:"-"`
	Type      ErrorType `json:"-"`
	Exposable bool      `json:"-"`
}

func (e *Error) Error() string {
	msg := strings.Join(e.Messages, " - ")
	if e.Cause != nil {
		msg = fmt.Sprintf("%s - Cause: ", e.Error())
	}
	return msg
}

func New(message string, t ErrorType, exposable bool) *Error {
	return &Error{
		Messages:  []string{message},
		Type:      t,
		Exposable: exposable,
	}
}

func Newf(message string, t ErrorType, exposable bool, params ...interface{}) *Error {
	return &Error{
		Messages:  []string{fmt.Sprintf(message, params...)},
		Type:      t,
		Exposable: exposable,
	}
}

func Wrap(e error, message string, t ErrorType, exposable bool) *Error {
	return &Error{
		Messages:  []string{message},
		Type:      t,
		Exposable: exposable,
		Cause:     e,
	}
}

func Wrapf(e error, message string, t ErrorType, exposable bool, params ...interface{}) *Error {
	return &Error{
		Messages:  []string{fmt.Sprintf(message, params...)},
		Type:      t,
		Exposable: exposable,
		Cause:     e,
	}
}
