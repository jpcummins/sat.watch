package http

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"time"

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

	// Create auth
	auth := smtp.PlainAuth("", formData.Username, formData.Password, formData.Host)

	// Create a custom dialer with timeout
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	// Try to connect to the SMTP server with timeout
	conn, err := dialer.Dial("tcp", fmt.Sprintf("%s:%d", formData.Host, formData.Port))
	if err != nil {
		return Render(c, http.StatusOK, templates.PageSettingsSmtp(sc.Config, &templates.SMTPTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to connect to SMTP server: %v", err),
		}, nil))
	}

	// Create SMTP client from the connection
	client, err := smtp.NewClient(conn, formData.Host)
	if err != nil {
		conn.Close()
		return Render(c, http.StatusOK, templates.PageSettingsSmtp(sc.Config, &templates.SMTPTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create SMTP client: %v", err),
		}, nil))
	}
	defer client.Close()

	// Start TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         formData.Host,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return Render(c, http.StatusOK, templates.PageSettingsSmtp(sc.Config, &templates.SMTPTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to start TLS: %v", err),
		}, nil))
	}

	// Try to authenticate
	if err := client.Auth(auth); err != nil {
		return Render(c, http.StatusOK, templates.PageSettingsSmtp(sc.Config, &templates.SMTPTestResult{
			Success: false,
			Message: fmt.Sprintf("Failed to authenticate with SMTP server: %v", err),
		}, nil))
	}

	// Close the connection
	if err := client.Quit(); err != nil {
		return Render(c, http.StatusOK, templates.PageSettingsSmtp(sc.Config, &templates.SMTPTestResult{
			Success: false,
			Message: fmt.Sprintf("Error closing SMTP connection: %v", err),
		}, nil))
	}

	// If we get here, the connection test was successful
	return c.Redirect(http.StatusSeeOther, "/app/settings")
}
