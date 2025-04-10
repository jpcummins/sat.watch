package http

import (
	"net/http"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/labstack/echo/v4"
)

type XpubController struct {
	API *api.API
}

func (xc XpubController) Index(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	xpub, err := xc.API.GetXpub(user.Model.ID, c.Param("xpub"))
	if err != nil {
		logger(c).Error().Err(err).Msg("GetXpub lookup failed")
		return Render(c, http.StatusInternalServerError, templates.PageXpubNotFound())
	}

	addresses := xc.API.GetAddressesForXpub(user.Model.ID, c.Param("xpub"))

	return Render(c, http.StatusOK, templates.PageXpub(xpub, addresses))
}

func (xc XpubController) Delete(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	err := xc.API.DeleteXpub(user.ID, c.Param("xpub"))
	if err != nil {
		logger(c).Error().Err(err).Msg("GetXpub delete failed")
		return Render(c, http.StatusInternalServerError, templates.PageXpubNotFound())
	}

	return c.Redirect(http.StatusSeeOther, "/app")
}
