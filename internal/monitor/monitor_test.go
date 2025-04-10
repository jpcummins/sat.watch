package monitor

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/jpcummins/go-electrum/electrum"
	"github.com/jpcummins/satwatch/internal/api"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockAddressAPI struct {
	mock.Mock
}

func (m *MockAddressAPI) GetAddresses() []api.Address {
	args := m.Called()
	return args.Get(0).([]api.Address)
}

type MockElectrumClient struct {
	mock.Mock
}

func (m *MockElectrumClient) GetTransaction(ctx context.Context, txHash string) (*electrum.GetTransactionResult, error) {
	args := m.Called(ctx, txHash)
	return args.Get(0).(*electrum.GetTransactionResult), args.Error(1)
}

func (m *MockElectrumClient) ListUnspent(ctx context.Context, scripthash string) ([]*electrum.ListUnspentResult, error) {
	args := m.Called(ctx, scripthash)
	return args.Get(0).([]*electrum.ListUnspentResult), args.Error(1)
}

func (m *MockElectrumClient) GetHistory(ctx context.Context, scripthash string) ([]*electrum.GetMempoolResult, error) {
	args := m.Called(ctx, scripthash)
	return args.Get(0).([]*electrum.GetMempoolResult), args.Error(1)
}

// Test helper functions
func createTestTransaction(inputs []TestInput, outputs []TestOutput, destinationAddress string) *wire.MsgTx {
	tx := wire.NewMsgTx(wire.TxVersion)

	// Add inputs if any
	for _, input := range inputs {
		hash, _ := chainhash.NewHashFromStr(input.Hash)
		outPoint := wire.NewOutPoint(hash, input.Index)
		txIn := wire.NewTxIn(outPoint, nil, nil)
		tx.AddTxIn(txIn)
	}

	// Add outputs if any
	for _, output := range outputs {
		var pkScript []byte

		// Handle special test cases
		if destinationAddress == "invalid_address" {
			// Create a deliberately invalid script for testing
			pkScript = []byte{0xFF, 0xFF, 0xFF, 0xFF} // Invalid script
		} else {
			// Create a proper P2PKH script
			addr, err := btcutil.DecodeAddress(destinationAddress, &chaincfg.MainNetParams)
			if err != nil {
				// For test cases that expect failure, return a transaction with an invalid script
				pkScript = []byte{0xFF, 0xFF, 0xFF, 0xFF}
			} else {
				pkScript, err = txscript.PayToAddrScript(addr)
				if err != nil {
					// For test cases that expect failure, return a transaction with an invalid script
					pkScript = []byte{0xFF, 0xFF, 0xFF, 0xFF}
				}
			}
		}
		tx.AddTxOut(wire.NewTxOut(output.Amount, pkScript))
	}

	return tx
}

type TestInput struct {
	Hash  string
	Index uint32
}

type TestOutput struct {
	Amount int64
}

func TestMonitorTransactions(t *testing.T) {
	// Enable debug logging for tests
	logger := zerolog.New(zerolog.NewTestWriter(t)).Level(zerolog.DebugLevel)

	tests := []struct {
		name           string
		watchedAddress api.Address
		tx             *wire.MsgTx
		isConfirmed    bool
		expectNotify   bool
		isSending      bool
		mockSetup      func(*MockAddressAPI, *MockElectrumClient) // New field for custom mock setup
	}{
		{
			name: "receiving_at_index_0",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-1",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash1",
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			),
			isConfirmed:  true,
			expectNotify: true,
			isSending:    false,
		},
		{
			name: "sending_from_index_0",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-2",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash2",
				UTXOs: []*electrum.ListUnspentResult{
					{
						Hash:     "0000000000000000000000000000000000000000000000000000000000000001",
						Position: 0,
					},
				},
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX",
			),
			isConfirmed:  true,
			expectNotify: true,
			isSending:    true,
		},
		{
			name: "receiving_unconfirmed",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-3",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash3",
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			),
			isConfirmed:  false,
			expectNotify: true,
			isSending:    false,
		},
		{
			name: "receiving_confirmed",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-4",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash4",
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			),
			isConfirmed:  true,
			expectNotify: true,
			isSending:    false,
		},
		{
			name: "sending_unconfirmed",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-5",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash5",
				UTXOs: []*electrum.ListUnspentResult{
					{
						Hash:     "0000000000000000000000000000000000000000000000000000000000000001",
						Position: 0,
					},
				},
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX",
			),
			isConfirmed:  false,
			expectNotify: true,
			isSending:    true,
		},
		{
			name: "sending_confirmed",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-6",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash6",
				UTXOs: []*electrum.ListUnspentResult{
					{
						Hash:     "0000000000000000000000000000000000000000000000000000000000000001",
						Position: 0,
					},
				},
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX",
			),
			isConfirmed:  true,
			expectNotify: true,
			isSending:    true,
		},
		{
			name: "sending_multiple_inputs",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-7",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash7",
				UTXOs: []*electrum.ListUnspentResult{
					{
						Hash:     "0000000000000000000000000000000000000000000000000000000000000001",
						Position: 0,
					},
					{
						Hash:     "0000000000000000000000000000000000000000000000000000000000000002",
						Position: 0,
					},
				},
			},
			tx: createTestTransaction(
				[]TestInput{
					{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0},
					{Hash: "0000000000000000000000000000000000000000000000000000000000000002", Index: 0},
				},
				[]TestOutput{{Amount: 100000}},
				"12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX",
			),
			isConfirmed:  true,
			expectNotify: true,
			isSending:    true,
		},
		{
			name: "error_tx_not_found",
			watchedAddress: api.Address{
				Model: api.Model{
					ID: "test-addr-8",
				},
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash_error1",
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			),
			isConfirmed:  true,
			expectNotify: false,
			isSending:    false,
			mockSetup: func(addressAPI *MockAddressAPI, electrumClient *MockElectrumClient) {
				// Return the watched address for both receiving and sending checks
				watchedAddr := api.Address{
					Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
					Scripthash: "scripthash_error1",
				}
				addressAPI.On("GetAddresses").Return([]api.Address{watchedAddr}).Times(2)
				// Return error for transaction lookup with empty but non-nil result
				electrumClient.On("GetTransaction", mock.Anything, mock.Anything).Return(
					&electrum.GetTransactionResult{}, fmt.Errorf("transaction not found"),
				).Once()
			},
		},
		{
			name: "error_invalid_input_index",
			watchedAddress: api.Address{
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash_error2",
				UTXOs: []*electrum.ListUnspentResult{
					{
						Hash:     "0000000000000000000000000000000000000000000000000000000000000001",
						Position: 0,
						Value:    100000,
						Height:   100,
					},
				},
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 999}}, // Invalid index
				[]TestOutput{{Amount: 90000}},
				"12c6DSiU4Rq3P4ZxziKxzrL5LmMBrzjrJX",
			),
			isConfirmed:  true,
			expectNotify: false,
			isSending:    true,
		},
		{
			name: "error_malformed_address",
			watchedAddress: api.Address{
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash_error3",
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{{Amount: 100000}},
				"invalid_address", // This will cause script creation to fail
			),
			isConfirmed:  true,
			expectNotify: false,
			isSending:    false,
		},
		{
			name: "error_empty_inputs",
			watchedAddress: api.Address{
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash_error4",
			},
			tx: createTestTransaction(
				[]TestInput{}, // Empty inputs
				[]TestOutput{{Amount: 100000}},
				"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			),
			isConfirmed:  true,
			expectNotify: false,
			isSending:    false,
		},
		{
			name: "error_empty_outputs",
			watchedAddress: api.Address{
				Address:    "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
				Scripthash: "scripthash_error5",
			},
			tx: createTestTransaction(
				[]TestInput{{Hash: "0000000000000000000000000000000000000000000000000000000000000001", Index: 0}},
				[]TestOutput{}, // Empty outputs
				"1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa",
			),
			isConfirmed:  true,
			expectNotify: false,
			isSending:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mocks
			mockAddressAPI := new(MockAddressAPI)
			mockElectrumClient := new(MockElectrumClient)

			// Setup expectations
			if tt.mockSetup != nil {
				tt.mockSetup(mockAddressAPI, mockElectrumClient)
			} else {
				mockAddressAPI.On("GetAddresses").Return([]api.Address{tt.watchedAddress}).Maybe()
				mockElectrumClient.On("GetTransaction", mock.Anything, mock.Anything).Return(
					&electrum.GetTransactionResult{
						Confirmations: int32(map[bool]int{true: 1, false: 0}[tt.isConfirmed]),
					},
					nil,
				).Maybe()
			}

			// Create monitor with proper initialization
			monitor := &TxMonitor{
				txStreams: make([]chan TxNotification, 0),
			}

			// Create a channel to receive notifications
			notifications := make(chan TxNotification, 1)
			monitor.txStreams = append(monitor.txStreams, notifications)

			// Create a done channel for cleanup
			done := make(chan bool)
			defer close(done)

			// Skip transaction creation for malformed address test
			var hexTx string
			if tt.name == "error_malformed_address" {
				hexTx = "invalid_tx_hex"
			} else {
				// Encode transaction to hex
				var buf bytes.Buffer
				tt.tx.Serialize(&buf)
				hexTx = hex.EncodeToString(buf.Bytes())
			}

			// Process the transaction
			rawTxChan := make(chan []string, 1)
			go func() {
				watchRawTx(logger, rawTxChan, monitor, mockAddressAPI, mockElectrumClient)
				done <- true
			}()

			// Send the transaction
			rawTxChan <- []string{"dummy", hexTx}
			close(rawTxChan)

			// Wait for notification or timeout
			select {
			case notification := <-notifications:
				assert.True(t, tt.expectNotify, "Expected no notification but got one")
				if tt.expectNotify {
					assert.Equal(t, tt.isConfirmed, notification.Confirmed)
					assert.Equal(t, tt.isSending, notification.Sent)
					assert.Equal(t, tt.watchedAddress.Address, notification.MatchedAddress[0].Address)
				}
			case <-time.After(100 * time.Millisecond):
				assert.False(t, tt.expectNotify, "Expected notification but got none")
			}

			// Wait for goroutine to finish
			<-done

			// Verify mock expectations
			mockAddressAPI.AssertExpectations(t)
			mockElectrumClient.AssertExpectations(t)
		})
	}
}
