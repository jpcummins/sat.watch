package http

import (
	"net/http"

	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/labstack/echo/v4"
)

type SMTPController struct {
	Config *configs.Config
}

func (sc SMTPController) Index(c echo.Context) error {
	return c.Render(http.StatusOK, "page.smtp", nil)
}

func (sc SMTPController) Update(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}

func (sc SMTPController) Test(c echo.Context) error {
	return c.NoContent(http.StatusNotImplemented)
}
