package http

import (
	"net/http"

	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/labstack/echo/v4"
)

type SMTPController struct {
	Config *configs.Config
}

type SMTPFormData struct {
	Host     string `form:"host"`
	Port     int    `form:"port"`
	Username string `form:"username"`
	Password string `form:"password"`
}

func (sc SMTPController) Index(c echo.Context) error {
	return Render(c, http.StatusOK, templates.PageSettingsSmtp(sc.Config, nil, nil))
}

func (sc SMTPController) Update(c echo.Context) error {
	var formData SMTPFormData
	if err := c.Bind(&formData); err != nil {
		return Render(c, http.StatusBadRequest, templates.PageSettingsSmtp(sc.Config, nil, err))
	}
	return Render(c, http.StatusOK, templates.PageSettingsSmtp(sc.Config, nil, nil))
}
