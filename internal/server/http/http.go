package http

import (
	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/clients"
	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/jpcummins/satwatch/internal/server/zmq"
)

func Init(api *api.API, electrumClient *electrum.Client, mockZmqServer *zmq.MockZmqServer, emailClient EmailClient, config configs.Config, bitcoinClient clients.BitcoinClient) {
	e := NewRouter(api, electrumClient, mockZmqServer, emailClient, config, bitcoinClient)
	host := ":" + config.Port
	e.Logger.Fatal(e.Start(host))
}
