package master

import (
	"net/http"
	"strings"

	"github.com/edgestore/edgestore/internal/errors"
	"github.com/edgestore/edgestore/internal/server"
)

func ER(err error) *server.ErrorResponse {
	code := http.StatusInternalServerError
	msg := err.Error()

	e, ok := err.(*errors.Error)
	if ok {
		slices := strings.Split(msg, ": ")
		msg = slices[len(slices)-1]
		switch e.Kind {
		case errors.Duplicate:
			code = http.StatusBadRequest
		case errors.Invalid:
			code = http.StatusBadRequest
		case errors.NotFound:
			code = http.StatusNotFound
		case errors.Permission:
			code = http.StatusUnauthorized
		}
	}

	return &server.ErrorResponse{
		Code:    code,
		Message: msg,
	}
}
