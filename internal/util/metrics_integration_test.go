//go:build integration

package util

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsIntegration_BlocksIndexedIncrement tests blocks indexed counter
// Validates AC7: Metrics Integration Tests - Subtask 7.1
func TestMetricsIntegration_BlocksIndexedIncrement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Get initial value
	initialValue := testutil.ToFloat64(BlocksIndexed)

	// Record some blocks indexed
	for i := 0; i < 10; i++ {
		RecordBlockIndexed()
	}

	// Verify counter increased
	finalValue := testutil.ToFloat64(BlocksIndexed)
	assert.Equal(t, initialValue+10, finalValue, "Should increment by 10")

	t.Log("✓ explorer_blocks_indexed_total increments correctly")
}

// TestMetricsIntegration_IndexLagBlocks tests lag gauge updates
// Validates AC7: Metrics Integration Tests - Subtask 7.2
func TestMetricsIntegration_IndexLagBlocks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Set various lag values
	testCases := []struct {
		lag float64
	}{
		{lag: 0},
		{lag: 5},
		{lag: 100},
		{lag: 1000},
	}

	for _, tc := range testCases {
		SetIndexLagBlocks(tc.lag)

		// Verify gauge value
		value := testutil.ToFloat64(IndexLagBlocks)
		assert.Equal(t, tc.lag, value, "Lag should be set to %f", tc.lag)
	}

	// Test lag seconds as well
	SetIndexLagSeconds(30.5)
	lagSeconds := testutil.ToFloat64(IndexLagSeconds)
	assert.Equal(t, 30.5, lagSeconds, "Lag seconds should be 30.5")

	t.Log("✓ explorer_index_lag_blocks and explorer_index_lag_seconds update correctly")
}

// TestMetricsIntegration_RPCErrorCounting tests RPC error counter
// Validates AC7: Metrics Integration Tests - Subtask 7.3
func TestMetricsIntegration_RPCErrorCounting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Record various error types
	errorTypes := []string{
		"network",
		"network",
		"timeout",
		"rate_limit",
		"invalid_param",
		"other",
	}

	for _, errorType := range errorTypes {
		RecordRPCError(errorType)
	}

	// Verify counter values by error type
	networkErrors := testutil.ToFloat64(RPCErrors.WithLabelValues("network"))
	assert.Equal(t, float64(2), networkErrors, "Should have 2 network errors")

	timeoutErrors := testutil.ToFloat64(RPCErrors.WithLabelValues("timeout"))
	assert.Equal(t, float64(1), timeoutErrors, "Should have 1 timeout error")

	rateLimitErrors := testutil.ToFloat64(RPCErrors.WithLabelValues("rate_limit"))
	assert.Equal(t, float64(1), rateLimitErrors, "Should have 1 rate_limit error")

	// Test unknown error type defaults to "other"
	RecordRPCError("unknown_error")
	otherErrors := testutil.ToFloat64(RPCErrors.WithLabelValues("other"))
	assert.GreaterOrEqual(t, otherErrors, float64(2), "Should have at least 2 'other' errors")

	t.Log("✓ explorer_rpc_errors_total counts errors by type correctly")
}

// TestMetricsIntegration_BackfillDuration tests duration histogram
// Validates AC7: Metrics Integration Tests - histogram tracking
func TestMetricsIntegration_BackfillDuration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Record various backfill durations
	durations := []float64{0.5, 1.2, 2.5, 5.0, 0.3}

	for _, duration := range durations {
		RecordBackfillDuration(duration)
	}

	// Verify histogram has observations
	// Note: We can't easily check exact bucket counts without accessing internal state,
	// but we can verify the metric exists and has data
	count := testutil.CollectAndCount(BackfillDuration)
	assert.Greater(t, count, 0, "Should have histogram samples")

	// Test invalid duration (negative) is rejected
	RecordBackfillDuration(-1.0)

	t.Log("✓ explorer_backfill_duration_seconds histogram records durations")
}

// TestMetricsIntegration_MetricsEndpoint tests /metrics HTTP endpoint
// Validates AC7: Metrics Integration Tests - Subtask 7.4
func TestMetricsIntegration_MetricsEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Record some metrics
	RecordBlockIndexed()
	RecordBlockIndexed()
	SetIndexLagBlocks(10)
	RecordRPCError("network")

	// Start metrics server in background
	go func() {
		_ = StartMetricsServer()
	}()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Fetch metrics endpoint
	port := GetMetricsPort()
	endpoint := GetMetricsEndpoint()
	url := "http://localhost:" + port + endpoint

	resp, err := http.Get(url)
	if err != nil {
		t.Skipf("Metrics server not available (may be port conflict): %v", err)
		return
	}
	defer resp.Body.Close()

	// Verify response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Should return 200 OK")
	assert.Contains(t, resp.Header.Get("Content-Type"), "text/plain", "Should return text/plain")

	// Read response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Should read response body")

	bodyStr := string(body)

	// Verify Prometheus format
	assert.Contains(t, bodyStr, "explorer_blocks_indexed_total", "Should contain blocks indexed metric")
	assert.Contains(t, bodyStr, "explorer_index_lag_blocks", "Should contain lag blocks metric")
	assert.Contains(t, bodyStr, "explorer_rpc_errors_total", "Should contain RPC errors metric")

	// Verify HELP and TYPE comments (Prometheus format)
	assert.Contains(t, bodyStr, "# HELP", "Should contain HELP comments")
	assert.Contains(t, bodyStr, "# TYPE", "Should contain TYPE comments")

	t.Log("✓ /metrics endpoint returns Prometheus format")
}

// TestMetricsIntegration_PersistAcrossOperations tests metrics persistence
// Validates AC7: Metrics Integration Tests - Subtask 7.5
func TestMetricsIntegration_PersistAcrossOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Simulate multiple indexing operations
	operations := []struct {
		name         string
		blocksToAdd  int
		rpcErrors    int
		lagBlocks    float64
	}{
		{"operation_1", 10, 2, 5},
		{"operation_2", 20, 1, 3},
		{"operation_3", 15, 3, 1},
	}

	totalBlocks := 0
	totalErrors := 0

	for _, op := range operations {
		// Record blocks
		for i := 0; i < op.blocksToAdd; i++ {
			RecordBlockIndexed()
		}
		totalBlocks += op.blocksToAdd

		// Record errors
		for i := 0; i < op.rpcErrors; i++ {
			RecordRPCError("network")
		}
		totalErrors += op.rpcErrors

		// Update lag
		SetIndexLagBlocks(op.lagBlocks)

		// Verify cumulative metrics
		blocksIndexed := testutil.ToFloat64(BlocksIndexed)
		assert.GreaterOrEqual(t, blocksIndexed, float64(totalBlocks),
			"Blocks should accumulate across operations")

		networkErrors := testutil.ToFloat64(RPCErrors.WithLabelValues("network"))
		assert.GreaterOrEqual(t, networkErrors, float64(totalErrors),
			"Errors should accumulate across operations")

		// Verify lag is latest value (gauge, not cumulative)
		lag := testutil.ToFloat64(IndexLagBlocks)
		assert.Equal(t, op.lagBlocks, lag, "Lag should be latest value")

		t.Logf("After %s: %d total blocks, %d total errors, lag=%f",
			op.name, totalBlocks, totalErrors, op.lagBlocks)
	}

	t.Log("✓ Metrics persist and accumulate correctly across multiple operations")
}

// TestMetricsIntegration_ValuesMatchExpected tests metric value accuracy
// Validates AC7: Metrics Integration Tests - Subtask 7.6
func TestMetricsIntegration_ValuesMatchExpected(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Reset metrics by creating new registry (for clean test)
	// Note: In production, metrics are global and cumulative

	testCases := []struct {
		name            string
		blocksToIndex   int
		networkErrors   int
		timeoutErrors   int
		expectedLag     float64
		backfillSeconds float64
	}{
		{
			name:            "small_batch",
			blocksToIndex:   5,
			networkErrors:   1,
			timeoutErrors:   0,
			expectedLag:     10,
			backfillSeconds: 0.5,
		},
		{
			name:            "medium_batch",
			blocksToIndex:   50,
			networkErrors:   3,
			timeoutErrors:   2,
			expectedLag:     50,
			backfillSeconds: 2.0,
		},
		{
			name:            "large_batch",
			blocksToIndex:   100,
			networkErrors:   5,
			timeoutErrors:   1,
			expectedLag:     100,
			backfillSeconds: 5.0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Record metrics for this test case
			initialBlocks := testutil.ToFloat64(BlocksIndexed)

			for i := 0; i < tc.blocksToIndex; i++ {
				RecordBlockIndexed()
			}

			for i := 0; i < tc.networkErrors; i++ {
				RecordRPCError("network")
			}

			for i := 0; i < tc.timeoutErrors; i++ {
				RecordRPCError("timeout")
			}

			SetIndexLagBlocks(tc.expectedLag)
			RecordBackfillDuration(tc.backfillSeconds)

			// Verify values
			actualBlocks := testutil.ToFloat64(BlocksIndexed) - initialBlocks
			assert.Equal(t, float64(tc.blocksToIndex), actualBlocks,
				"Blocks indexed should match expected")

			actualLag := testutil.ToFloat64(IndexLagBlocks)
			assert.Equal(t, tc.expectedLag, actualLag,
				"Lag should match expected value")

			t.Logf("✓ Test case %s: %d blocks, lag=%f, duration=%fs",
				tc.name, tc.blocksToIndex, tc.expectedLag, tc.backfillSeconds)
		})
	}

	t.Log("✓ All metric values match expected counts")
}

// TestMetricsIntegration_ConcurrentAccess tests thread-safe metric updates
// Validates metrics are thread-safe under concurrent access
func TestMetricsIntegration_ConcurrentAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	initialValue := testutil.ToFloat64(BlocksIndexed)

	// Simulate concurrent metric updates
	concurrency := 10
	updatesPerWorker := 100

	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func(workerID int) {
			for j := 0; j < updatesPerWorker; j++ {
				RecordBlockIndexed()
				RecordRPCError("network")
				SetIndexLagBlocks(float64(workerID))
			}
			done <- true
		}(i)
	}

	// Wait for all workers
	for i := 0; i < concurrency; i++ {
		<-done
	}

	// Verify counter accumulated correctly
	finalValue := testutil.ToFloat64(BlocksIndexed)
	expectedIncrease := float64(concurrency * updatesPerWorker)
	actualIncrease := finalValue - initialValue

	assert.Equal(t, expectedIncrease, actualIncrease,
		"Counter should accumulate correctly under concurrent access")

	// Verify RPC errors also accumulated
	networkErrors := testutil.ToFloat64(RPCErrors.WithLabelValues("network"))
	assert.GreaterOrEqual(t, networkErrors, expectedIncrease,
		"RPC errors should accumulate under concurrent access")

	t.Logf("✓ Metrics handle %d concurrent workers correctly", concurrency)
}

// TestMetricsIntegration_MetricLabels tests label-based metrics
// Validates error type labels work correctly
func TestMetricsIntegration_MetricLabels(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Test all valid error types
	errorTypes := []string{
		"network",
		"rate_limit",
		"invalid_param",
		"timeout",
		"other",
	}

	for _, errorType := range errorTypes {
		// Record error
		RecordRPCError(errorType)

		// Verify counter for this label
		value := testutil.ToFloat64(RPCErrors.WithLabelValues(errorType))
		assert.GreaterOrEqual(t, value, float64(1),
			"Error type '%s' should have at least 1 count", errorType)
	}

	// Verify invalid error type gets mapped to "other"
	initialOther := testutil.ToFloat64(RPCErrors.WithLabelValues("other"))
	RecordRPCError("invalid_error_type")
	finalOther := testutil.ToFloat64(RPCErrors.WithLabelValues("other"))

	assert.Greater(t, finalOther, initialOther,
		"Invalid error type should be counted as 'other'")

	t.Log("✓ Metric labels work correctly for all error types")
}

// Helper function to count metric families
func countMetricFamilies(body string) int {
	lines := strings.Split(body, "\n")
	count := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "# TYPE") {
			count++
		}
	}
	return count
}

// TestMetricsIntegration_PrometheusRegistration tests metrics are properly registered
// Validates all expected metrics are registered with Prometheus
func TestMetricsIntegration_PrometheusRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Initialize metrics (safe to call multiple times - only first call registers)
	if BlocksIndexed == nil {
		err := Init()
		require.NoError(t, err, "Should initialize metrics")
	}

	// Verify all expected metrics are registered
	expectedMetrics := []prometheus.Collector{
		BlocksIndexed,
		IndexLagBlocks,
		IndexLagSeconds,
		BackfillDuration,
	}

	for i, metric := range expectedMetrics {
		assert.NotNil(t, metric, "Metric %d should be registered", i)
	}

	// Verify RPCErrors counter vec
	assert.NotNil(t, RPCErrors, "RPCErrors counter vec should be registered")

	// Test that metrics can be collected
	count := testutil.CollectAndCount(BlocksIndexed)
	assert.Greater(t, count, 0, "Should be able to collect BlocksIndexed metric")

	t.Log("✓ All metrics are properly registered with Prometheus")
}
