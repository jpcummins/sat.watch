package api

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jpcummins/go-electrum/electrum"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockUtxoResult implements the UtxoResult interface for testing.
type mockUtxoResult struct {
	scriptHash string
	utxos      []*electrum.ListUnspentResult
}

func (r mockUtxoResult) GetScriptHash() string {
	return r.scriptHash
}

func (r mockUtxoResult) GetUtxoData() []*electrum.ListUnspentResult {
	return r.utxos
}

// mockBatchResults provides a minimal implementation of pgx.BatchResults for testing.
type mockBatchResults struct{}

func (mbr mockBatchResults) Exec() (pgconn.CommandTag, error) { return pgconn.CommandTag{}, nil }
func (mbr mockBatchResults) Query() (pgx.Rows, error)         { return nil, nil }
func (mbr mockBatchResults) QueryRow() pgx.Row                { return nil }
func (mbr mockBatchResults) Close() error                     { return nil } // Important: Needs to close without error

// mockDB implements the DB interface for testing.
type mockDB struct{}

func (m *mockDB) Ping(ctx context.Context) error {
	return nil // Assume ping succeeds
}

func (m *mockDB) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return mockBatchResults{} // Return our simple mock results
}

func (m *mockDB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	// For this specific race test, Select might not be called, or we don't need
	// it to populate data. Return nil to indicate success.
	return nil
}

// Exec implements the DB interface for testing.
func (m *mockDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	// Return a default CommandTag and no error for testing purposes.
	return pgconn.NewCommandTag(""), nil
}

// Query implements the DB interface for testing.
func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	// Return nil Rows and no error. Tests needing row data would require a more sophisticated mock.
	return nil, nil // Need to return a concrete type that satisfies pgx.Rows, like pgxmock.Rows
}

// QueryRow implements the DB interface for testing.
func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	// Return nil Row. Tests needing row data would require a more sophisticated mock.
	return nil // Need to return a concrete type that satisfies pgx.Row
}

// mockUtxoMonitor implements the UtxoMonitor interface for testing.
type mockUtxoMonitor struct {
	resultChan chan interface{}
	mu         sync.Mutex
}

// newMockUtxoMonitor creates a monitor that immediately sends results.
func newMockUtxoMonitor(bufferSize int) *mockUtxoMonitor {
	return &mockUtxoMonitor{
		resultChan: make(chan interface{}, bufferSize),
	}
}

func (m *mockUtxoMonitor) EnqueueScan(scriptHash string) {
	// Simulate immediate scan completion by sending a result right away.
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resultChan <- mockUtxoResult{
		scriptHash: scriptHash,
		utxos: []*electrum.ListUnspentResult{
			{Value: 1000}, // Sample UTXO
		},
	}
}

func (m *mockUtxoMonitor) GetUtxoStream() <-chan interface{} {
	return m.resultChan
}

// TestCreateAddressRaceCondition verifies that UTXOs are updated even if
// the UtxoMonitor returns results immediately after EnqueueScan is called.
func TestCreateAddressRaceCondition(t *testing.T) {
	// Arrange
	monitor := newMockUtxoMonitor(1)

	apiInstance := &API{
		monitor:   monitor,
		addresses: []Address{},
		db:        &mockDB{},
	}

	// Start the background goroutine that processes UTXO updates.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Read the single result we expect from the mock monitor.
		select {
		case result := <-monitor.GetUtxoStream():
			utxoResult, ok := result.(UtxoResult)
			require.True(t, ok, "Result should be UtxoResult type")
			err := apiInstance.UpdateAddressUTXOs(utxoResult.GetScriptHash(), utxoResult.GetUtxoData())
			assert.NoError(t, err)
		case <-time.After(1 * time.Second): // Timeout if no result received
			t.Error("Timeout waiting for UTXO update")
		}
	}()

	testAddr := Address{
		Model:   Model{ID: uuid.NewString()},
		UserID:  uuid.NewString(),
		Address: "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq",
	}

	// Act
	err := apiInstance.CreateAddress(testAddr)
	require.NoError(t, err)

	// Wait for the background update goroutine to finish processing.
	wg.Wait()
}
