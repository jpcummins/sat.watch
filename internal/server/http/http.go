package http

import (
	"net/http"
	"time"

	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/clients"
	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/jpcummins/satwatch/internal/server/zmq"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func Init(api *api.API, electrumClient *electrum.Client, mockZmqServer *zmq.MockZmqServer, emailClient EmailClient, config *configs.Config, bitcoinClient clients.BitcoinClient) (*echo.Echo, error) {
	log.Info().Msg("starting webserver")
	e := NewRouter(api, electrumClient, mockZmqServer, emailClient, config, bitcoinClient)
	host := ":" + config.Port

	serverErrors := make(chan error, 1)

	go func() {
		if err := e.Start(host); err != nil && err != http.ErrServerClosed {
			e.Logger.Errorf("Failed to start server: %v", err)
			serverErrors <- err
		}
		close(serverErrors)
	}()

	select {
	case err := <-serverErrors:
		return nil, err
	case <-time.After(100 * time.Millisecond):
		log.Info().Str("host", host).Msg("finished initializing webserver")
		return e, nil
	}
}
