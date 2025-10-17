//go:build integration
// +build integration

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/cmd"
	"github.com/zondax/fil-trace-check/internal"
)

// TestValidateNullBlocksCmd_Integration tests the actual command execution with S3 and RPC
// Run with: go test -tags=integration -v -run TestValidateNullBlocksCmd_Integration
func TestValidateNullBlocksCmd_Integration(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test DB
	tempDir, err := os.MkdirTemp("", "nullblocks_cmd_integration")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create the command
	command := cmd.ValidateNullBlocksCmd()
	command.SetContext(t.Context())

	// Set test flags
	testStart := testData.ValidateNullBlocks.Start
	testEnd := testData.ValidateNullBlocks.End

	err = command.Flags().Set(internal.StartFlag, fmt.Sprint(testStart))
	require.NoError(t, err)

	err = command.Flags().Set(internal.EndFlag, fmt.Sprint(testEnd))
	require.NoError(t, err)

	err = command.Flags().Set(internal.DBPathFlag, tempDir)
	require.NoError(t, err)

	// Run the command
	t.Logf("Running NullBlocks validation from height %d to %d", testStart, testEnd)
	err = command.RunE(command, []string{})
	require.NoError(t, err)

	// Verify DB was created and contains progress
	dbPath := filepath.Join(tempDir, internal.NullBlocksCheck+".db")
	assert.FileExists(t, dbPath, "Database file should exist")

	// Open DB to check results
	db, err := api.NewDB(tempDir, internal.NullBlocksCheck)
	require.NoError(t, err)
	defer db.Close()

	// Check latest height was recorded
	latestHeight, err := db.GetLatestHeight()
	require.NoError(t, err)
	assert.Equal(t, int64(testEnd), latestHeight, "Latest height should match end height")

	assertResults(t, db)
}
