package clients

import (
	"context"
	"crypto/tls"
	"strconv"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/configs"
)

func NewElectrumClient(cfg configs.Config) (*electrum.Client, error) {
	log.Info().Msg("initializing electrum")
	ctx := context.TODO()
	host := cfg.ElectrumHost + ":" + strconv.Itoa(cfg.ElectrumPort)

	logger := log.With().Str("module", "electrum").Str("host", host).Logger()
	logger.Info().Msg("connecting")

	var client *electrum.Client
	var err error

	if cfg.ElectrumSSL {
		config := tls.Config{InsecureSkipVerify: true}
		client, err = electrum.NewClientSSL(ctx, host, &config)
	} else {
		client, err = electrum.NewClientTCP(ctx, host)
	}

	if err != nil {
		logger.Err(err)
		return nil, err
	}

	serverVersion, serverProtocolVersion, err := client.ServerVersion(ctx)
	if err != nil {
		logger.Err(err)
		return nil, err
	}

	logger.Info().
		Str("version", serverVersion).
		Str("protocol", serverProtocolVersion).
		Msg("connected")

	// Making sure connection is not closed with timed "client.ping" call
	go func() {
		for {
			if err := client.Ping(ctx); err != nil {
				logger.Err(err)
			}
			time.Sleep(60 * time.Second)
			logger.Debug().Msg("ping")
		}
	}()

	go func(c *electrum.Client) {
		for {
			select {
			case err := <-c.Error:
				logger.Err(err)
			}
		}
	}(client)

	log.Info().Msg("fininished initializing electrum")
	return client, nil
}
