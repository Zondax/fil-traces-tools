package internal

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal/types"
)

func TestUpdateProgressHeight(t *testing.T) {
	tests := []struct {
		name    string
		height  int64
		success bool
		message string
	}{
		{
			name:    "successful progress update",
			height:  12345,
			success: true,
			message: "ok",
		},
		{
			name:    "failed progress update",
			height:  67890,
			success: false,
			message: "error occurred",
		},
		{
			name:    "zero height",
			height:  0,
			success: true,
			message: "processed",
		},
		{
			name:    "negative height",
			height:  -1,
			success: false,
			message: "invalid height",
		},
		{
			name:    "large height",
			height:  999999999,
			success: true,
			message: "ok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary database
			tmpDir := t.TempDir()

			db, err := api.NewDB(tmpDir, "test-bucket")
			require.NoError(t, err)
			defer func() {
				require.NoError(t, db.Close())
			}()

			// Test UpdateProgressHeight
			UpdateProgressHeight(tt.height, tt.success, tt.message, db)

			// Verify the data was stored correctly
			data, err := db.GetAllKVAsJSON()
			require.NoError(t, err)

			// Parse the JSON to verify the content
			result := make(map[string]types.Progress)
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			key := strconv.FormatInt(tt.height, 10)
			progress, exists := result[key]
			require.True(t, exists)
			assert.Equal(t, tt.success, progress.Success)
			assert.Equal(t, tt.message, progress.Message)
		})
	}
}

func TestUpdateProgressAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		height  int64
		success bool
		message string
	}{
		{
			name:    "successful address progress",
			address: "f1234",
			height:  100,
			success: true,
			message: "ok",
		},
		{
			name:    "failed address progress",
			address: "f5678",
			height:  200,
			success: false,
			message: "balance mismatch",
		},
		{
			name:    "empty address",
			address: "",
			height:  300,
			success: false,
			message: "invalid address",
		},
		{
			name:    "long address",
			address: "f1234567890abcdefghijklmnopqrstuvwxyz",
			height:  400,
			success: true,
			message: "processed",
		},
		{
			name:    "special characters in message",
			address: "f9999",
			height:  500,
			success: false,
			message: "error: invalid state at height: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary database
			tmpDir := t.TempDir()

			db, err := api.NewDB(tmpDir, "test-bucket")
			require.NoError(t, err)
			defer func() {
				require.NoError(t, db.Close())
			}()

			// Test UpdateProgressAddress
			UpdateProgressAddress(tt.address, tt.height, tt.success, tt.message, db)

			// Verify the data was stored correctly
			data, err := db.GetAllKVAsJSON()
			require.NoError(t, err)

			// Parse the JSON to verify the content
			result := make(map[string]types.Progress)
			err = json.Unmarshal(data, &result)
			require.NoError(t, err)

			key := tt.address + "_" + strconv.FormatInt(tt.height, 10)
			progress, exists := result[key]
			require.True(t, exists)
			assert.Equal(t, tt.success, progress.Success)
			assert.Equal(t, tt.message, progress.Message)
		})
	}
}

func TestUpdateProgressMultipleEntries(t *testing.T) {
	// Create a temporary database
	tmpDir := t.TempDir()

	db, err := api.NewDB(tmpDir, "test-bucket")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	// Add multiple progress entries
	entries := []struct {
		isHeight bool
		address  string
		height   int64
		success  bool
		message  string
	}{
		{isHeight: true, height: 100, success: true, message: "ok"},
		{isHeight: true, height: 200, success: false, message: "error"},
		{isHeight: false, address: "f1234", height: 100, success: true, message: "balance ok"},
		{isHeight: false, address: "f1234", height: 200, success: false, message: "balance error"},
		{isHeight: false, address: "f5678", height: 100, success: true, message: "processed"},
	}

	// Insert all entries
	for _, entry := range entries {
		if entry.isHeight {
			UpdateProgressHeight(entry.height, entry.success, entry.message, db)
		} else {
			UpdateProgressAddress(entry.address, entry.height, entry.success, entry.message, db)
		}
	}

	// Verify all entries were stored
	data, err := db.GetAllKVAsJSON()
	require.NoError(t, err)

	result := make(map[string]types.Progress)
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)

	// Check that we have all entries
	assert.Equal(t, 5, len(result))

	// Verify specific entries
	assert.Equal(t, true, result["100"].Success)
	assert.Equal(t, "ok", result["100"].Message)

	assert.Equal(t, false, result["200"].Success)
	assert.Equal(t, "error", result["200"].Message)

	assert.Equal(t, true, result["f1234_100"].Success)
	assert.Equal(t, "balance ok", result["f1234_100"].Message)

	assert.Equal(t, false, result["f1234_200"].Success)
	assert.Equal(t, "balance error", result["f1234_200"].Message)

	assert.Equal(t, true, result["f5678_100"].Success)
	assert.Equal(t, "processed", result["f5678_100"].Message)
}
