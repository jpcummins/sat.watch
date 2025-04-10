package http

import (
	"net/http"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/labstack/echo/v4"
)

type UnauthController struct {
}

func getUser(c echo.Context) *api.User {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		return nil
	}
	return &user
}

func (ac UnauthController) Home(c echo.Context) error {
	return c.Redirect(http.StatusFound, "/login")
}
