package http

import (
	"errors"
	"net/http"
	"strings"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

const keyUser = "user"

type AuthController struct {
	Config *configs.Config
	API    *api.API
}

func (ac AuthController) GetLogin(c echo.Context) error {
	return Render(c, http.StatusOK, templates.PageLogin(nil))
}

func (ac AuthController) Login(c echo.Context) error {
	username := c.FormValue("username")
	password := c.FormValue("password")
	rememberMe := c.FormValue("rememberMe") == "on"

	if username == "" || password == "" {
		return Render(c, http.StatusUnauthorized, templates.PageLogin(errors.New("Username and password are required")))
	}

	user, err := ac.API.GetUserByUsername(username)
	if err != nil {
		return Render(c, http.StatusUnauthorized, templates.PageLogin(errors.New("Invalid credentials")))
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return Render(c, http.StatusUnauthorized, templates.PageLogin(errors.New("Invalid credentials")))
	}

	sess, err := session.Get(sessionKey, c)
	if err != nil {
		logger(c).Err(err).Msg("unable to get session")
		return Render(c, http.StatusInternalServerError, templates.PageLogin(errors.New("An unexpected error occurred")))
	}

	if rememberMe {
		sess.Options.MaxAge = 30 * 24 * 60 * 60 // 30 days in seconds
	}

	sess.Values[keyUser] = user.ID
	if err := sess.Save(c.Request(), c.Response()); err != nil {
		logger(c).Err(err).Msg("unable to save session")
		return Render(c, http.StatusInternalServerError, templates.PageLogin(errors.New("An unexpected error occurred")))
	}

	redirect := c.QueryParam("redirect")
	if strings.HasPrefix(redirect, "/app/settings/email/") {
		return c.Redirect(http.StatusSeeOther, redirect)
	}

	return c.Redirect(http.StatusSeeOther, "/app")
}

func (ac AuthController) Logout(c echo.Context) error {
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
