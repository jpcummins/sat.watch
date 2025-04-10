package http

import (
	"context"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

func Render(ctx echo.Context, statusCode int, t templ.Component) error {
	buf := templ.GetBuffer()
	defer templ.ReleaseBuffer(buf)

	nonce := ctx.Get("csp-nonce").(string)
	cspContext := context.WithValue(ctx.Request().Context(), "csp-nonce", nonce)

	version := ctx.Get("app-version").(string)
	versionContext := context.WithValue(cspContext, "app-version", version)

	cspContext = templ.WithNonce(cspContext, nonce)

	if err := t.Render(versionContext, buf); err != nil {
		return err
	}

	return ctx.HTML(statusCode, buf.String())
}
