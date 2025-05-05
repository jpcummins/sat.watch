package http

import (
	"crypto/sha512"
	"fmt"
	"net/http"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/sessions"
	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/clients"
	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/jpcummins/satwatch/internal/lib"
	"github.com/jpcummins/satwatch/internal/server/zmq"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
)

const sessionKey = "s"

var validate *validator.Validate

func NewRouter(api *api.API, electrumClient *electrum.Client, mockZmqServer *zmq.MockZmqServer, emailClient EmailClient, config *configs.Config, bitcoinClient clients.BitcoinClient) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	hash := sha512.Sum512([]byte(config.Secret))
	cookieStore := sessions.NewCookieStore(hash[:64])
	cookieStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   0,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	e.Use(session.Middleware(cookieStore))
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.Gzip())
	e.Use(middleware.SecureWithConfig(middleware.SecureConfig{
		HSTSMaxAge:         31536000,
		XFrameOptions:      "DENY",
		ContentTypeNosniff: "nosniff",
		ReferrerPolicy:     "same-origin",
	}))
	e.Pre(middleware.RemoveTrailingSlash())

	permissionsPolicyMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			nonce, err := lib.GenerateRandomStringURLSafe(24)
			if err != nil {
				log.Fatal().Err(err).Msg("Unable to create nonce")
			}

			c.Set("csp-nonce", nonce)
			c.Set("app-version", config.Version)
			c.Response().Header().Set("Content-Security-Policy", fmt.Sprintf("default-src 'none'; frame-ancestors 'none'; base-uri 'none'; form-action sat.watch 'self'; manifest-src 'self'; script-src 'nonce-%s'; style-src 'self'; img-src 'self' data:; connect-src wss://sat.watch 'self'; frame-src https://www.youtube-nocookie.com;", nonce))
			c.Response().Header().Set("Permissions-Policy", "autoplay=(), camera=(), cross-origin-isolated=(), display-capture=(), encrypted-media=(), fullscreen=(), geolocation=(), gyroscope=(), keyboard-map=(), magnetometer=(), microphone=(), midi=(), payment=(), picture-in-picture=(), screen-wake-lock=(), sync-xhr=(), usb=(), xr-spatial-tracking=(), clipboard-read=(), clipboard-write=(), gamepad=()")
			return next(c)
		}
	}
	e.Use(permissionsPolicyMiddleware)

	validate = validator.New(validator.WithRequiredStructEnabled())
	logger := log.With().Str("module", "echo").Logger()

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogLatency:   true,
		LogRemoteIP:  true,
		LogHost:      true,
		LogMethod:    true,
		LogURI:       true,
		LogRoutePath: true,
		LogRequestID: true,
		LogReferer:   true,
		LogUserAgent: true,
		LogStatus:    true,
		LogError:     true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			fields := map[string]interface{}{
				"uri":        v.URI,
				"method":     v.Method,
				"status":     v.Status,
				"route_path": v.RoutePath,
				"request_id": v.RequestID,
				"latency":    v.Latency,
				"remote_ip":  v.RemoteIP,
				"referer":    v.Referer,
				"user_agent": v.UserAgent,
			}

			switch {
			case v.Status >= 500:
				logger.Error().Fields(fields).Err(v.Error).Msg("server error")
			case v.Status >= 400:
				logger.Warn().Fields(fields).Err(v.Error).Msg("client error")
			case v.Status >= 200 && v.Status < 400:
				logger.Info().Fields(fields).Err(v.Error).Msg("success")
			default:
				logger.Error().Fields(fields).Err(v.Error).Msg("unexpected status code")
			}
			return nil
		},
	}))

	e.File("/static/js/flowbite.min.js", "./ts/node_modules/flowbite/dist/flowbite.min.js")
	e.File("/android-chrome-192x192.png", "./internal/server/http/web/static/android-chrome-192x192.png")
	e.File("/android-chrome-512x512.png", "./internal/server/http/web/static/android-chrome-512x512.png")
	e.File("/apple-touch-icon.png", "./internal/server/http/web/static/apple-touch-icon.png")
	e.File("/browserconfig.xml", "./internal/server/http/web/static/browserconfig.xml")
	e.File("/favicon-16x16.png", "./internal/server/http/web/static/favicon-16x16.png")
	e.File("/favicon-32x32.png", "./internal/server/http/web/static/favicon-32x32.png")
	e.File("/favicon.ico", "./internal/server/http/web/static/favicon.ico")
	e.File("/mstile-150x150.png", "./internal/server/http/web/static/mstile-150x150.png")
	e.File("/safari-pinned-tab.svg", "./internal/server/http/web/static/safari-pinned-tab.svg")
	e.File("/site.webmanifest", "./internal/server/http/web/static/site.webmanifest")

	e.Add(http.MethodGet, "/static/*", echo.StaticDirectoryHandler(os.DirFS("./internal/server/http/web/static"), false))

	unauth := UnauthController{}
	e.GET("/", unauth.Home, unauthMiddleware(api))

	authController := AuthController{
		Config: config,
		API:    api,
	}
	e.GET("/login", authController.GetLogin, unauthMiddleware(api))
	e.POST("/login", authController.Login, unauthMiddleware(api))
	e.GET("/app/logout", authController.Logout)

	g := e.Group("/app")
	g.Use(authMiddleware(api))

	appController := AppController{api, config}
	g.GET("", appController.Home)

	settingsController := SettingsController{
		API:    api,
		URL:    config.URL,
		Config: config,
	}
	g.GET("/settings", settingsController.Index)
	g.POST("/settings/deleteAccount", settingsController.DeleteAccount)

	notifyController := NotificationController{
		API:         api,
		EmailClient: emailClient,
	}
	g.GET("/settings/email/create", notifyController.NewEmail)
	g.POST("/settings/email/create", notifyController.CreateEmail)
	g.GET("/settings/email/:notification/edit", notifyController.EditEmail)
	g.POST("/settings/email/:notification/edit", notifyController.UpdateEmail)
	g.GET("/settings/email/:notification/verify", notifyController.Verify)
	g.POST("/settings/email/:notification/verify", notifyController.ResetVerification)
	g.POST("/settings/email/:notification/delete", notifyController.DeleteEmail)

	g.GET("/settings/webhooks/create", notifyController.NewWebhook)
	g.POST("/settings/webhooks/create", notifyController.CreateWebhook)
	g.GET("/settings/webhooks/:notification/edit", notifyController.EditWebhook)
	g.POST("/settings/webhooks/:notification/edit", notifyController.UpdateWebhook)
	g.POST("/settings/webhooks/:notification/delete", notifyController.DeleteWebhook)

	smtpController := SMTPController{
		Config: config,
	}
	smtpGroup := g.Group("/settings/smtp")
	smtpGroup.Use(requireAdminMiddleware)
	smtpGroup.GET("", smtpController.Index)
	smtpGroup.POST("", smtpController.Update)

	addressController := AddressController{
		API:            api,
		Config:         config,
		Gap:            config.Gap,
		EmailClient:    emailClient,
		ElectrumClient: electrumClient,
		BitcoinClient:  bitcoinClient,
	}
	g.GET("/addresses/:address", addressController.Index)
	g.GET("/addresses/create", addressController.New)
	g.POST("/addresses/create", addressController.Create)
	g.POST("/addresses/:address/delete", addressController.Delete)
	g.GET("/addresses/status", addressController.Status)

	xpubController := XpubController{
		API: api,
	}
	g.GET("/xpubs/:xpub", xpubController.Index)
	g.POST("/xpubs/:xpub/delete", xpubController.Delete)

	if mockZmqServer != nil {
		mockZmqController := MockZmqController{
			MockZmqServer: *mockZmqServer,
		}
		g.GET("/mockzmq", mockZmqController.New)
		g.POST("/mockzmq", mockZmqController.Create)
	}

	return e
}
