package http

import (
	"net/http"

	"github.com/jpcummins/satwatch/internal/server/http/web/templates"
	"github.com/jpcummins/satwatch/internal/server/zmq"
	"github.com/labstack/echo/v4"
)

type MockZmqController struct {
	MockZmqServer zmq.MockZmqServer
}

func (mc MockZmqController) New(c echo.Context) error {
	return Render(c, http.StatusOK, templates.PageMockZmq())
}

func (mc MockZmqController) Create(c echo.Context) error {
	type Params struct {
		RawTx string `form:"rawtx" binding:"required"`
	}

	var params Params
	if err := c.Bind(&params); err != nil {
		logger(c).Warn().Err(err).Msg("unable to parse tx")
		return Render(c, http.StatusUnprocessableEntity, templates.PageMockZmq())
	}

	mc.MockZmqServer.SendTx(params.RawTx)
	return c.Redirect(http.StatusSeeOther, "/app/mockzmq")
}
