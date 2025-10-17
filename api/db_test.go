package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDB(t *testing.T) {
	tests := []struct {
		name        string
		bucket      string
		expectError bool
	}{
		{
			name:        "valid database creation",
			bucket:      "test-bucket",
			expectError: false,
		},
		{
			name:        "empty bucket name",
			bucket:      "",
			expectError: true,
		},
		{
			name:        "long bucket name",
			bucket:      "very-long-bucket-name-with-many-characters",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			db, err := NewDB(tmpDir, tt.bucket)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, db)
				assert.Equal(t, tt.bucket, db.bucket)
				require.NoError(t, db.Close())
			}
		})
	}

	t.Run("invalid path", func(t *testing.T) {
		db, err := NewDB("/invalid/path/that/does/not/exist/test.db", "bucket")
		assert.Error(t, err)
		assert.Nil(t, db)
	})
}

func TestDB_Insert(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir, "test-bucket")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	tests := []struct {
		name   string
		key    string
		data   interface{}
		hasErr bool
	}{
		{
			name: "insert string data",
			key:  "key1",
			data: "string value",
		},
		{
			name: "insert struct data",
			key:  "key2",
			data: struct {
				Name  string
				Value int
			}{Name: "test", Value: 42},
		},
		{
			name: "insert map data",
			key:  "key3",
			data: map[string]interface{}{
				"field1": "value1",
				"field2": 123,
			},
		},
		{
			name:   "insert with empty key",
			key:    "",
			data:   "test",
			hasErr: true,
		},
		{
			name: "insert nil data",
			key:  "key4",
			data: nil,
		},
		{
			name: "overwrite existing key",
			key:  "key1",
			data: "new value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := db.Insert(tt.key, tt.data)
			if tt.hasErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDB_GetLatestHeight(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir, "test-bucket")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	// Test empty bucket
	height, err := db.GetLatestHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(0), height)

	// Insert some height-based keys
	testData := struct{ Value string }{Value: "test"}
	err = db.Insert("100", testData)
	require.NoError(t, err)

	err = db.Insert("200", testData)
	require.NoError(t, err)

	height, err = db.GetLatestHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(200), height)

	err = db.Insert("50", testData)
	require.NoError(t, err)

	height, err = db.GetLatestHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(50), height)

	err = db.Insert("f0014_130", testData)
	require.NoError(t, err)

	height, err = db.GetLatestHeight()
	assert.NoError(t, err)
	assert.Equal(t, int64(50), height)
}

func TestDB_GetAllKVAsJSON(t *testing.T) {
	tmpDir := t.TempDir()

	db, err := NewDB(tmpDir, "test-bucket")
	require.NoError(t, err)
	defer func() {
		require.NoError(t, db.Close())
	}()

	// Test empty bucket
	data, err := db.GetAllKVAsJSON()
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Empty(t, result)

	// Insert various data types
	testData := []struct {
		key   string
		value interface{}
	}{
		{
			key: "string-key",
			value: map[string]interface{}{
				"type": "string",
				"data": "hello world",
			},
		},
		{
			key: "number-key",
			value: map[string]interface{}{
				"type":  "number",
				"value": 42,
			},
		},
		{
			key: "complex-key",
			value: map[string]interface{}{
				"nested": map[string]interface{}{
					"field1": "value1",
					"field2": []int{1, 2, 3},
				},
			},
		},
	}

	// Insert all test data
	for _, td := range testData {
		err := db.Insert(td.key, td.value)
		require.NoError(t, err)
	}

	// Get all data as JSON
	data, err = db.GetAllKVAsJSON()
	require.NoError(t, err)

	// Parse and verify
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Equal(t, len(testData), len(result))

	// Verify each entry
	for _, td := range testData {
		value, exists := result[td.key]
		assert.True(t, exists)
		assert.NotNil(t, value)
	}
}
