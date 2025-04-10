package main

import (
	"database/sql"
	"embed"

	"github.com/bnkamalesh/errors"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
	"github.com/rs/zerolog/log"

	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/actions"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/jpcummins/satwatch/internal/clients"
	"github.com/jpcummins/satwatch/internal/configs"
	"github.com/jpcummins/satwatch/internal/monitor"
	"github.com/jpcummins/satwatch/internal/server/http"
	"github.com/jpcummins/satwatch/internal/server/zmq"
)

type ServerDependencies struct {
	api            *api.API
	electrumClient *electrum.Client
	mockZmqServer  *zmq.MockZmqServer
	email          clients.MailClient
	config         configs.Config
	bitcoinClient  clients.BitcoinClient
}

//go:embed db/migrations/*.sql
var embedMigrations embed.FS

func initDependencies() (*ServerDependencies, error) {
	log.Info().Msg("starting sat.watch")
	cfg, err := configs.InitConifg("internal/configs/config.yml")
	if err != nil {
		return nil, errors.InternalErr(err, "Unable to initialize config")
	}

	err = migratedb(cfg.DatabaseUrl)
	if err != nil {
		return nil, errors.InternalErr(err, "DB migration failed")
	}

	electrumClient, err := clients.NewElectrumClient(cfg)
	if err != nil {
		return nil, errors.InternalErr(err, "Unable to initialize electrum client")
	}

	utxoMonitor := monitor.InitUtxoMonitor(electrumClient)

	bitcoinClient, err := clients.NewBitcoinClient(cfg.RPCHost, cfg.RPCUser, cfg.RPCPassword, cfg.Gap)
	if err != nil {
		return nil, errors.InternalErr(err, "Unable to initialize bitcoin client")
	}

	api, err := api.Init(cfg, utxoMonitor)
	if err != nil {
		return nil, errors.InternalErr(err, "Unable to initialize API client")
	}

	emailClient, err := clients.NewMailClient(api, cfg.URL, cfg.SmtpHost, cfg.SmtpPort, cfg.SmtpUser, cfg.SmtpPassword)
	if err != nil {
		return nil, errors.InternalErr(err, "Unable to initialize mail client")
	}

	var mockZmqServer *zmq.MockZmqServer
	if cfg.Environment == "development" {
		log.Info().Str("init", "MockZmqServer").Msg("Starting MockZmqServer")
		mockZmqServer, err = zmq.Init(cfg.ZMQMockHost, cfg.ZMQMockPort)
		if err != nil {
			return nil, errors.InternalErr(err, "Unable to initialize mock zmq server")
		}
		mockTxMonitor, err := monitor.InitTxMonitor(cfg.ZMQMockHost, cfg.ZMQMockPort, api, electrumClient)
		if err != nil {
			return nil, errors.InternalErr(err, "Unable to initialize mock tx monitor")
		}
		actions.InitWebhookNotifier(api, mockTxMonitor)
		actions.InitEmailNotifier(api, mockTxMonitor, emailClient)
	}

	txMonitor, err := monitor.InitTxMonitor(cfg.ZMQHost, cfg.ZMQPort, api, electrumClient)
	if err != nil {
		return nil, errors.InternalErr(err, "Unable to initialize tx monitor")
	}

	actions.InitWebhookNotifier(api, txMonitor)
	actions.InitEmailNotifier(api, txMonitor, emailClient)

	return &ServerDependencies{api, electrumClient, mockZmqServer, emailClient, cfg, bitcoinClient}, nil
}

func migratedb(dburl string) error {
	db, err := sql.Open("postgres", dburl)
	if err != nil {
		return err
	}

	defer db.Close()

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	if err := goose.Up(db, "db/migrations"); err != nil {
		return err
	}

	return nil
}

func main() {
	deps, err := initDependencies()
	if err != nil {
		log.Fatal().Err(err).Msg("App initialization failed.")
	}

	log.Info().Msg("starting webserver")
	http.Init(deps.api, deps.electrumClient, deps.mockZmqServer, deps.email, deps.config, deps.bitcoinClient)
}
