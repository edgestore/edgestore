package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (er *ErrorResponse) Error() string {
	return fmt.Sprintf("%d - %s", er.Code, er.Message)
}

// Abort is a helper function that calls `Abort()` and then `JSON` internally.
// This method stops the chain, writes the status code and return a JSON body with HTTP status code and error message.
// It also sets the Content-Type as "application/json".
func Abort(c *gin.Context, code int, message string) {
	c.AbortWithStatusJSON(code, &ErrorResponse{
		Code:    code,
		Message: message,
	})
}
