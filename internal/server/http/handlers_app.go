package http

import (
	"net/http"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/labstack/echo/v4"
)

type AppController struct {
	API *api.API
}

func (ac AppController) Home(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	xpubs, _ := ac.API.GetXpubs(user.ID)
	addresses := ac.API.GetAddressesForUser(user.ID)
	webhooks, _ := ac.API.GetUserWebhooks(user.ID)
	emails, _ := ac.API.GetUserEmails(user.ID)

	return Render(c, http.StatusOK, templates.PageApp(xpubs, addresses, webhooks, emails))
}
