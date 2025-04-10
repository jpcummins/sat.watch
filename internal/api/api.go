package api

import (
	"context"
	"encoding/gob"
	"fmt"
	"sync"
	"time"

	"github.com/bnkamalesh/errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/configs"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/rs/zerolog/log"
)

type API struct {
	db        *pgxpool.Pool
	addresses []Address
	mu        sync.Mutex
	monitor   UtxoMonitor
}

type Model struct {
	ID        string
	CreatedAt *time.Time `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

type UtxoResult interface {
	GetScriptHash() string
	GetUtxoData() []*electrum.ListUnspentResult
}

type UtxoMonitor interface {
	EnqueueScan(scriptHash string)
	GetUtxoStream() <-chan interface{}
}

func Init(config configs.Config, utxoMonitor UtxoMonitor) (*API, error) {
	logger := log.With().Str("module", "api").Logger()
	logger.Info().Msg("initializing")

	api := API{}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, config.DatabaseUrl)
	if err != nil {
		return &api, errors.InternalErr(err, "Unable to create pg pool")
	}

	api.db = db
	api.monitor = utxoMonitor

	err = db.Ping(ctx)
	if err != nil {
		return &api, errors.InternalErr(err, "Unable to connect to db")
	}

	var addresses []Address
	if err := pgxscan.Select(ctx, db, &addresses, "SELECT id, created_at, updated_at, user_id, xpub_id, address, scripthash, name, is_external, address_index FROM addresses WHERE deleted_at IS NULL"); err != nil {
		logger.Err(err)
		return &api, errors.InternalErrf(err, "Unable to query db %s", config.DatabaseUrl)
	}

	addressCount := len(addresses)
	for i, address := range addresses {
		logger.Debug().Msg(fmt.Sprintf("Getting UTXOs for %d of %d (%f%%)", i+1, addressCount, 100*float32(i+1)/float32(addressCount)))
		utxoMonitor.EnqueueScan(address.Scripthash)
	}

	go api.updateUTXOs(utxoMonitor)

	gob.Register(User{})

	api.mu.Lock()
	defer api.mu.Unlock()
	api.addresses = addresses
	logger.Info().Msg("finished initializing")
	return &api, err
}

func (api *API) updateUTXOs(monitor UtxoMonitor) {
	for utxoResult := range monitor.GetUtxoStream() {

		utxoResult, ok := utxoResult.(UtxoResult)
		if !ok {
			log.Fatal().Msg("unable to typecast utxo result")
		}

		api.UpdateAddressUTXOs(utxoResult.GetScriptHash(), utxoResult.GetUtxoData())
	}
}
