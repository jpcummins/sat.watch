package http

import (
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func logger(c echo.Context) *zerolog.Logger {
	l := log.With().
		Str("module", "echo").
		Str("request_id", c.Response().Header().Get(echo.HeaderXRequestID)).
		Logger()
	return &l
}
