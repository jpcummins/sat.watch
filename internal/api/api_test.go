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

type mockBatchResults struct{}

func (mbr mockBatchResults) Exec() (pgconn.CommandTag, error) { return pgconn.CommandTag{}, nil }
func (mbr mockBatchResults) Query() (pgx.Rows, error)         { return nil, nil }
func (mbr mockBatchResults) QueryRow() pgx.Row                { return nil }
func (mbr mockBatchResults) Close() error                     { return nil }

type mockDB struct{}

func (m *mockDB) Ping(ctx context.Context) error {
	return nil
}

func (m *mockDB) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return mockBatchResults{}
}

func (m *mockDB) Select(ctx context.Context, dest interface{}, query string, args ...interface{}) error {
	return nil
}

func (m *mockDB) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), nil
}

func (m *mockDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}

func (m *mockDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return nil
}

type mockUtxoMonitor struct {
	resultChan chan interface{}
	mu         sync.Mutex
}

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

func TestCreateAddressRaceCondition(t *testing.T) {
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

	err := apiInstance.CreateAddress(testAddr)
	require.NoError(t, err)

	wg.Wait()
}
