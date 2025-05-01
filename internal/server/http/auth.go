package http

import (
	"net/http"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

func authMiddleware(api *api.API) func(echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get(sessionKey, c)
			if err != nil {
				logger(c).Err(err).Msg("unable to get session")
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			}

			userId, ok := sess.Values[keyUser].(string)
			if !ok {
				logger(c).Error().Msg("invalid user")
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			}

			user, err := api.GetUser(userId)
			if err != nil {
				logger(c).Error().Msg("unable to get user")
				return c.Redirect(http.StatusTemporaryRedirect, "/login")
			}

			c.Set(keyUser, user)
			return next(c)
		}
	}
}

func unauthMiddleware(api *api.API) func(echo.HandlerFunc) echo.HandlerFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sess, err := session.Get(sessionKey, c)
			if err != nil {
				return next(c)
			}

			userId, ok := sess.Values[keyUser].(string)
			if !ok {
				return next(c)
			}

			user, err := api.GetUser(userId)
			if err != nil {
				return next(c)
			}

			c.Set(keyUser, user)
			return next(c)
		}
	}
}

func requireAdminMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		user, ok := c.Get(keyUser).(api.User)
		if !ok || !user.IsAdmin {
			return c.NoContent(http.StatusUnauthorized)
		}
		return next(c)
	}
}
