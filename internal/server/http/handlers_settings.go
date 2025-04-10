package http

import (
	"errors"
	"net/http"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

type SettingsController struct {
	API *api.API
	URL string
}

func (sc SettingsController) Index(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	webhooks, err := sc.API.GetUserWebhooks(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("webhooks not found")
		return Render(c, http.StatusInternalServerError, templates.PageSettings(user, nil, nil, errors.New("Internal error")))
	}

	emails, err := sc.API.GetUserEmails(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("emails not found")
		return Render(c, http.StatusInternalServerError, templates.PageSettings(user, webhooks, nil, errors.New("Internal error")))
	}
	return Render(c, http.StatusOK, templates.PageSettings(user, webhooks, emails, nil))
}

func (sc SettingsController) DeleteAccount(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	err := sc.API.SoftDeleteUser(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("Unable to delete user")
		return c.NoContent(http.StatusInternalServerError)
	}

	err = sc.API.DeleteUserXpubs(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("Unable to delete xpubs")
		return c.NoContent(http.StatusInternalServerError)
	}

	err = sc.API.DeleteAddresses(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("Unable to delete addresses")
		return c.NoContent(http.StatusInternalServerError)
	}

	err = sc.API.DeleteEmails(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("Unable to delete emails")
		return c.NoContent(http.StatusInternalServerError)
	}

	err = sc.API.DeleteWebhooks(user.ID)
	if err != nil {
		logger(c).Error().Err(err).Msg("Unable to delete webhooks")
		return c.NoContent(http.StatusInternalServerError)
	}

	sess, err := session.Get(sessionKey, c)
	if err != nil {
		logger(c).Error().Err(err).Msg("unable to get session")
		return c.Redirect(http.StatusSeeOther, "/login")
	}

	sess.Options.MaxAge = -1

	if err := sess.Save(c.Request(), c.Response()); err != nil {
		logger(c).Err(err).Msg("unable to save session")
	}

	return c.Redirect(http.StatusSeeOther, "/login")
}
