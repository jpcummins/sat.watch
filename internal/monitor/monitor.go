// Package monitor provides Bitcoin transaction monitoring functionality.
// It watches for new transactions via ZMQ and tracks transactions related to specified addresses.
package monitor

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/jpcummins/go-bitcoin"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// BitcoinLogger wraps zerolog.Logger to provide Bitcoin-specific logging functionality.
type BitcoinLogger struct {
	log zerolog.Logger
}

// newLogger creates a new BitcoinLogger instance with module-specific context.
func newLogger() *BitcoinLogger {
	return &BitcoinLogger{
		log: log.With().Str("module", "txmonitor").Logger(),
	}
}

func (l *BitcoinLogger) Debugf(format string, args ...interface{}) {
	f := fmt.Sprintf(strings.TrimSuffix(format, "\n"), args...)
	l.log.Debug().Msg(f)
}

func (l *BitcoinLogger) Infof(format string, args ...interface{}) {
	f := fmt.Sprintf(strings.TrimSuffix(format, "\n"), args...)
	l.log.Info().Msg(f)
}

func (l *BitcoinLogger) Warnf(format string, args ...interface{}) {
	f := fmt.Sprintf(strings.TrimSuffix(format, "\n"), args...)
	l.log.Warn().Msg(f)
}

func (l *BitcoinLogger) Errorf(format string, args ...interface{}) {
	f := fmt.Sprintf(strings.TrimSuffix(format, "\n"), args...)
	l.log.Error().Msg(f)
}

func (l *BitcoinLogger) Fatalf(format string, args ...interface{}) {
	f := fmt.Sprintf(strings.TrimSuffix(format, "\n"), args...)
	l.log.Fatal().Msg(f)
}

// TxMonitor handles the monitoring of Bitcoin transactions.
// It maintains a list of subscribers that receive transaction notifications.
type TxMonitor struct {
	sync.Mutex
	txStreams []chan TxNotification
}

// TxNotification represents a notification about a relevant Bitcoin transaction.
type TxNotification struct {
	MatchedAddress []api.Address // Addresses that matched this transaction
	Tx             wire.MsgTx    // The actual transaction
	Sent           bool          // Whether funds were sent from a watched address
	Confirmed      bool          // Whether the transaction is confirmed
	Amount         int           // Transaction amount in satoshis
}

// Subscribe creates and returns a new channel for receiving transaction notifications.
// The channel will receive notifications for all transactions matching watched addresses.
func (m *TxMonitor) Subscribe() <-chan TxNotification {
	txStream := make(chan TxNotification)
	m.Lock()
	defer m.Unlock()
	m.txStreams = append(m.txStreams, txStream)
	return txStream
}

// AddressAPI defines the interface for retrieving watched addresses.
type AddressAPI interface {
	GetAddresses() []api.Address
}

// InitTxMonitor creates and initializes a new TxMonitor instance.
// It sets up ZMQ subscription for raw transactions and starts the monitoring process.
func InitTxMonitor(host string, port int, addressApi AddressAPI, electrumClient ElectrumClient) (*TxMonitor, error) {
	logger := log.With().Str("module", "txmonitor").Logger()
	logger.Info().Str("host", host).Int("port", port).Msg("initializing tx monitor")

	logger.Info().Str("host", host).Int("port", port).Msg("connecting to zmq")
	zmq := bitcoin.NewZMQ(host, port, newLogger())
	monitor := TxMonitor{}
	internalTxStream := make(chan []string)

	go watchRawTx(logger, internalTxStream, &monitor, addressApi, electrumClient)
	done := make(chan bool)
	defer close(done)
	err := zmq.Subscribe("rawtx", internalTxStream, done)
	<-done

	logger.Info().Msg("finished initializing tx monitor")
	return &monitor, err
}

// decodeTx decodes a hex-encoded transaction into a wire.MsgTx.
func decodeTx(hexTx string) (*wire.MsgTx, error) {
	decoded, err := hex.DecodeString(hexTx)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex transaction: %w", err)
	}

	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(decoded))
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}

	return &tx, nil
}

// findMatchingOutputs checks transaction outputs for matches with watched addresses.
// Returns matched addresses and the total amount sent to them.
func findMatchingOutputs(tx *wire.MsgTx, addressApi AddressAPI, logger zerolog.Logger) ([]api.Address, int) {
	var matchedAddresses []api.Address
	amount := 0

	for _, outTx := range tx.TxOut {
		// Extract addresses from the output script
		_, addresses, _, err := txscript.ExtractPkScriptAddrs(outTx.PkScript, &chaincfg.MainNetParams)
		if err != nil {
			logger.Error().Err(err).Msg("could not extract addresses from output script")
			continue
		}

		// Check if any extracted address matches our watched addresses
		for _, outAddress := range addresses {
			outAddressStr := outAddress.String()
			for _, watchedAddress := range addressApi.GetAddresses() {
				if watchedAddress.Address == outAddressStr {
					logger.Debug().Str("address_id", watchedAddress.ID).Msg("found matching address")
					matchedAddresses = append(matchedAddresses, watchedAddress)
					amount = int(outTx.Value)
				}
			}
		}
	}

	return matchedAddresses, amount
}

func addressesToStrings(addresses []btcutil.Address) []string {
	result := make([]string, len(addresses))
	for i, addr := range addresses {
		result[i] = addr.String()
	}
	return result
}

// checkSentFunds checks if the transaction spends from any watched addresses.
// Returns true and the matched address if found.
func checkSentFunds(tx *wire.MsgTx, addressApi AddressAPI, logger zerolog.Logger) (bool, []api.Address) {
	var matchedAddresses []api.Address

	for _, inTx := range tx.TxIn {
		prvOutPoint := inTx.PreviousOutPoint
		for _, address := range addressApi.GetAddresses() {
			for _, utxo := range address.UTXOs {
				if utxo.Hash == prvOutPoint.Hash.String() && utxo.Position == prvOutPoint.Index {
					logger.Debug().
						Str("address_id", address.ID).
						Str("tx_hash", utxo.Hash).
						Msg("found spent UTXO from watched address")
					matchedAddresses = append(matchedAddresses, address)
				}
			}
		}
	}

	return len(matchedAddresses) > 0, matchedAddresses
}

// watchRawTx processes incoming raw transactions and notifies subscribers of relevant transactions.
func watchRawTx(logger zerolog.Logger, ch chan []string, monitor *TxMonitor, addressApi AddressAPI, electrumClient ElectrumClient) {
	logger.Debug().Msg("starting transaction monitoring")
	for rawTx := range ch {
		if len(rawTx) < 2 {
			logger.Error().Msg("received invalid transaction data")
			continue
		}

		tx, err := decodeTx(rawTx[1])
		if err != nil {
			logger.Error().Err(err).Msg("failed to decode transaction")
			continue
		}

		txLogger := logger.With().Str("tx_hash", tx.TxHash().String()).Logger()
		matchedAddresses, amount := findMatchingOutputs(tx, addressApi, txLogger)

		sentFunds, sentAddresses := checkSentFunds(tx, addressApi, txLogger)
		if sentFunds {
			matchedAddresses = append(matchedAddresses, sentAddresses...)
		}

		if len(matchedAddresses) > 0 {
			confirmed, err := isConfirmed(tx, electrumClient)
			if err != nil {
				txLogger.Error().Err(err).Msg("failed to check transaction confirmation status")
				continue
			}

			monitor.fanOutTx(TxNotification{
				Tx:             *tx,
				MatchedAddress: matchedAddresses,
				Amount:         amount,
				Sent:           sentFunds,
				Confirmed:      confirmed,
			})
		}
	}
}

// isConfirmed checks if a transaction has been confirmed on the blockchain.
func isConfirmed(tx *wire.MsgTx, electrumClient ElectrumClient) (bool, error) {
	txResult, err := electrumClient.GetTransaction(context.Background(), tx.TxHash().String())
	if err != nil {
		return false, fmt.Errorf("failed to get transaction details: %w", err)
	}

	return txResult.Confirmations > 0, nil
}

// fanOutTx sends a transaction notification to all subscribers.
func (m *TxMonitor) fanOutTx(txNotification TxNotification) {
	m.Lock()
	defer m.Unlock()
	for _, txStream := range m.txStreams {
		txStream <- txNotification
	}
}
