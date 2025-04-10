package http

import (
	"errors"
	"net/http"

	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/labstack/echo/v4"
)

type EmailClient interface {
	SendVerification(email api.Email) error
}

type NotificationController struct {
	API         *api.API
	EmailClient EmailClient
}

func (nc NotificationController) NewWebhook(c echo.Context) error {
	return Render(c, http.StatusOK, templates.PageNotificationNewWebhook(nil))
}

func (nc NotificationController) NewEmail(c echo.Context) error {
	return Render(c, http.StatusOK, templates.PageNotificationNewEmail(nil))
}

func (nc NotificationController) CreateEmail(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	type FormParams struct {
		Email       string `form:"email"`
		Description string `form:"description"`
		Pubkey      string `form:"pubkey"`
	}

	var params FormParams
	if err := c.Bind(&params); err != nil {
		logger(c).Warn().Err(err).Msg("Unable to parse")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewEmail(errors.New("Unable to process request.")))
	}

	if err := validate.Var(params.Email, "required,email"); err != nil {
		logger(c).Warn().Err(err).Msg("email parse error")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewEmail(errors.New("Invalid email address")))
	}
	if err := nc.API.CreateEmail(user.ID, params.Email, params.Description, params.Pubkey); err != nil {
		userError := "Unable to save email."

		if err.Error() == "Invalid Pubkey" {
			userError = "Invalid pubkey"
		}
		logger(c).Warn().Err(err).Msg("Unable to create email")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewEmail(errors.New(userError)))
	}
	email, err := nc.API.GetEmailByAddress(user.ID, params.Email)
	if err != nil {
		logger(c).Warn().Err(err).Msg("Unable get email")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewEmail(errors.New("Unable to get email")))
	}

	err = nc.EmailClient.SendVerification(email)
	if err != nil {
		logger(c).Warn().Err(err).Msg("Unable to send verification email")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewEmail(errors.New("Unable to send verification email")))
	}

	return c.Redirect(http.StatusSeeOther, "/app/settings/email/"+email.ID+"/verify")
}

func (nc NotificationController) CreateWebhook(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	type FormParams struct {
		Url  string `form:"url"`
		Name string `form:"name"`
	}

	var params FormParams
	if err := c.Bind(&params); err != nil {
		logger(c).Warn().Err(err).Msg("Unable to parse")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewWebhook(errors.New("Unable to process request.")))
	}

	if err := validate.Var(params.Url, "required,http_url"); err != nil {
		logger(c).Warn().Err(err).Msg("webhook url parse error")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewWebhook(errors.New("Invalid URL")))
	}
	if err := nc.API.CreateWebhook(user.ID, params.Name, params.Url); err != nil {
		logger(c).Warn().Err(err).Msg("Unable to create webhook")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationNewWebhook(errors.New("Unable to create webhook")))
	}

	return c.Redirect(http.StatusSeeOther, "/app/settings")
}

func (nc NotificationController) Verify(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	notification := c.Param("notification")
	email, err := nc.API.GetEmail(user.ID, notification)
	if err != nil {
		logger(c).Warn().Err(err).Msg("email lookup failed")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationVerify(nil, errors.New("Failed to lookup email")))
	}

	code := c.QueryParam("code")

	if code == "" {
		return Render(c, http.StatusOK, templates.PageNotificationVerify(&email, nil))
	}

	err = nc.API.VerifyEmail(user.ID, notification, code)
	if err != nil {
		logger(c).Warn().Err(err).Msg("email verification failed")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationVerify(nil, errors.New("email verification failed")))
	}

	return c.Redirect(http.StatusSeeOther, "/app/settings/email/"+notification+"/verify")
}

func (nc NotificationController) ResetVerification(c echo.Context) error {
	notification := c.Param("notification")

	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	email, err := nc.API.ResetVerificationCode(user.ID, notification)
	if err != nil {
		logger(c).Warn().Err(err).Msg("Unable reset email")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationVerify(nil, errors.New("Unable reset verification")))
	}

	err = nc.EmailClient.SendVerification(email)
	if err != nil {
		logger(c).Warn().Err(err).Msg("Unable to send verification email")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationVerify(nil, errors.New("Unable to send verification email")))
	}
	return c.Redirect(http.StatusSeeOther, "/app/settings")
}

func (nc NotificationController) EditWebhook(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	webhook, err := nc.API.GetWebhook(user.Model.ID, c.Param("notification"))
	if err != nil {
		logger(c).Error().Err(err).Msg("webhook db lookup failed")
		return Render(c, http.StatusInternalServerError, templates.PageNotificationEditWebhook(nil, nil))
	}

	return Render(c, http.StatusOK, templates.PageNotificationEditWebhook(&webhook, nil))
}

func (nc NotificationController) EditEmail(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	email, err := nc.API.GetEmail(user.Model.ID, c.Param("notification"))
	if err != nil {
		logger(c).Error().Err(err).Msg("email db lookup failed")
		return Render(c, http.StatusInternalServerError, templates.PageNotificationEdit(nil, nil))
	}

	return Render(c, http.StatusOK, templates.PageNotificationEditEmail(&email, nil))
}

func (nc NotificationController) UpdateWebhook(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	type FormParams struct {
		Url  string `form:"url"`
		Name string `form:"name"`
	}

	var params FormParams
	if err := c.Bind(&params); err != nil {
		logger(c).Warn().Err(err).Msg("Unable to parse")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationEditWebhook(nil, errors.New("Unable to process request.")))
	}

	err := nc.API.UpdateWebhook(user.ID, c.Param("notification"), params.Url, params.Name)
	if err != nil {
		logger(c).Warn().Err(err).Msg("Unable update webhook")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationEditWebhook(nil, errors.New("Unable to send verification email")))
	}

	return c.Redirect(http.StatusSeeOther, "/app/settings")
}

func (nc NotificationController) UpdateEmail(c echo.Context) error {
	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	type FormParams struct {
		Description string `form:"description"`
		Pubkey      string `form:"pubkey"`
	}

	var params FormParams
	if err := c.Bind(&params); err != nil {
		logger(c).Warn().Err(err).Msg("Unable to parse")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationEditEmail(nil, errors.New("Unable to process request.")))
	}

	if params.Description != "" {
		err := nc.API.UpdateEmailDescription(user.ID, c.Param("notification"), params.Description)
		if err != nil {
			logger(c).Warn().Err(err).Msg("Unable update email description")
			return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationEditEmail(nil, errors.New("Unable to send verification email")))
		}
	}

	if params.Pubkey != "" {
		err := nc.API.UpdateEmailPubkey(user.ID, c.Param("notification"), params.Pubkey)
		if err != nil {
			logger(c).Warn().Err(err).Msg("Unable update email pubkey")
			return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationEditEmail(nil, errors.New("Unable to send verification email")))
		}
	}

	return c.Redirect(http.StatusSeeOther, "/app/settings")
}

func (nc NotificationController) DeleteEmail(c echo.Context) error {
	notification := c.Param("notification")

	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	err := nc.API.DeleteEmail(user.ID, notification)
	if err != nil {
		logger(c).Warn().Err(err).Msg("Unable delete email")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationVerify(nil, errors.New("Unable delete verification")))
	}

	return c.Redirect(http.StatusSeeOther, "/app/settings")
}

func (nc NotificationController) DeleteWebhook(c echo.Context) error {
	notification := c.Param("notification")

	user, ok := c.Get(keyUser).(api.User)
	if !ok {
		logger(c).Error().Msg("unauthorized access")
		return c.Redirect(http.StatusUnauthorized, "/login")
	}

	err := nc.API.DeleteWebhook(user.ID, notification)
	if err != nil {
		logger(c).Warn().Err(err).Msg("Unable delete webhook")
		return Render(c, http.StatusUnprocessableEntity, templates.PageNotificationEditWebhook(nil, errors.New("Unable delete verification")))
	}

	return c.Redirect(http.StatusSeeOther, "/app/settings")
}
