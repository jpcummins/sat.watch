package monitor

import (
	"context"
	"time"

	"github.com/jpcummins/go-electrum/electrum"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ElectrumClient interface {
	ListUnspent(ctx context.Context, scripthash string) ([]*electrum.ListUnspentResult, error)
	GetHistory(ctx context.Context, scripthash string) ([]*electrum.GetMempoolResult, error)
	GetTransaction(ctx context.Context, txHash string) (*electrum.GetTransactionResult, error)
}

type UtxoResult struct {
	scriptHash string
	utxos      []*electrum.ListUnspentResult
}

func (ur UtxoResult) GetScriptHash() string {
	return ur.scriptHash
}

func (ur UtxoResult) GetUtxoData() []*electrum.ListUnspentResult {
	return ur.utxos
}

type UtxoMonitor struct {
	electrum      ElectrumClient
	log           zerolog.Logger
	addressStream chan string
	utxoStream    chan interface{}
}

func InitUtxoMonitor(electrumClient ElectrumClient) UtxoMonitor {
	monitor := UtxoMonitor{
		electrum:      electrumClient,
		log:           log.With().Str("module", "UtxoScanner").Logger(),
		addressStream: make(chan string, 1000),
		utxoStream:    make(chan interface{}),
	}

	go monitor.monitorUtxos()

	return monitor
}

func (um UtxoMonitor) monitorUtxos() {
	for scriptHash := range um.addressStream {
		um.log.Debug().Str("scriptHash", scriptHash).Msg("Calling ListUnspent")
		start := time.Now()

		utxos, err := um.electrum.ListUnspent(context.TODO(), scriptHash)
		if err != nil {
			um.log.Error().Err(err).Msg("Error calling electrum ListUnspent")
		}

		um.utxoStream <- UtxoResult{
			scriptHash: scriptHash,
			utxos:      utxos,
		}

		duration := time.Since(start)
		um.log.Debug().Str("scriptHash", scriptHash).Str("duration", duration.String()).Msg("finished")
	}
}

func (um UtxoMonitor) GetUtxoStream() <-chan interface{} {
	return um.utxoStream
}

func (um UtxoMonitor) EnqueueScan(scriptHash string) {
	um.log.Debug().Str("scriptHash", scriptHash).Msg("enqueued")
	um.addressStream <- scriptHash
}
