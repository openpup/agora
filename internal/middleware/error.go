package middleware

import (
	"github.com/cloudwego/hertz/pkg/app"

	pkgerrors "github.com/openpup/agora/internal/pkg/errors"
)

func writeError(c *app.RequestContext, status int, code, message string) {
	requestID, _ := c.Get(RequestIDKey)
	requestIDString, _ := requestID.(string)
	c.JSON(status, pkgerrors.ErrorResponse{
		Error: pkgerrors.ErrorBody{
			Code:      code,
			Message:   message,
			RequestID: requestIDString,
		},
	})
}
