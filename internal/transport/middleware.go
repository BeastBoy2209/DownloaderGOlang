package transport

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v5"
)

func RequestIDMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		reqID := c.Request().Header.Get("Request-Id")
		if reqID == "" {
			reqID = uuid.NewString()
		}

		c.Request().Header.Set("Request-Id", reqID)
		c.Response().Header().Set("Request-Id", reqID)

		return next(c)
	}
}
