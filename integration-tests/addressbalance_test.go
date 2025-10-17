//go:build integration
// +build integration

package tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/cmd"
	"github.com/zondax/fil-trace-check/internal"
)

// TestValidateAddressBalanceCmd_Integration tests the actual command execution with S3 and RPC
// Run with: go test -tags=integration -v -run TestValidateAddressBalanceCmd_Integration
func TestValidateAddressBalanceCmd_Integration(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test DB
	tempDir, err := os.MkdirTemp("", "addressbalance_cmd_integration")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// addresses
	testAddresses := testData.ValidateAddressBalance.Addresses
	// Create a test address file
	addressFile := filepath.Join(tempDir, "test_addresses.txt")
	testAddressesBytes := []byte(strings.Join(testAddresses, "\n"))
	err = os.WriteFile(addressFile, testAddressesBytes, 0644)
	require.NoError(t, err)

	// Create the command
	command := cmd.ValidateAddressBalanceCmd()
	command.SetContext(t.Context())

	// Set test flags
	err = command.Flags().Set(internal.AddressFileFlag, addressFile)
	require.NoError(t, err)

	err = command.Flags().Set(internal.DBPathFlag, tempDir)
	require.NoError(t, err)

	err = command.Flags().Set(internal.EventProviderFlag, "beryx")
	require.NoError(t, err)

	err = command.Flags().Set(internal.EventProviderTokenFlag, os.Getenv("BERYX_TOKEN"))
	require.NoError(t, err)

	// Run the command
	t.Logf("Running AddressBalance validation with address file: %s", addressFile)
	err = command.RunE(command, []string{})
	require.NoError(t, err)

	// Verify DB was created and contains progress
	dbPath := filepath.Join(tempDir, internal.AddressBalanceCheck+".db")
	assert.FileExists(t, dbPath, "Database file should exist")

	// Open DB to check results
	db, err := api.NewDB(tempDir, internal.AddressBalanceCheck)
	require.NoError(t, err)
	defer db.Close()

	assertResults(t, db)
}

// TestValidateAddressBalanceSequentialCmd_Integration tests the actual command execution with S3 and RPC
// Run with: go test -tags=integration -v -run TestValidateAddressBalanceSequentialCmd_Integration
func TestValidateAddressBalanceSequentialCmd_Integration(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test DB
	tempDir, err := os.MkdirTemp("", "addressbalance_sequential_cmd_integration")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// addresses
	testAddresses := testData.ValidateAddressBalanceSequential.Addresses
	// Create a test address file
	addressFile := filepath.Join(tempDir, "test_addresses.txt")
	testAddressesBytes := []byte(strings.Join(testAddresses, "\n"))
	err = os.WriteFile(addressFile, testAddressesBytes, 0644)
	require.NoError(t, err)

	// Create the command
	command := cmd.ValidateAddressBalanceSequentialCmd()
	command.SetContext(t.Context())

	// Set test flags
	err = command.Flags().Set(internal.AddressFileFlag, addressFile)
	require.NoError(t, err)

	err = command.Flags().Set(internal.DBPathFlag, tempDir)
	require.NoError(t, err)

	err = command.Flags().Set(internal.StartFlag, fmt.Sprint(testData.ValidateAddressBalanceSequential.Start))
	require.NoError(t, err)

	err = command.Flags().Set(internal.EndFlag, fmt.Sprint(testData.ValidateAddressBalanceSequential.End))
	require.NoError(t, err)

	// Run the command
	t.Logf("Running AddressBalance validation with address file: %s", addressFile)
	err = command.RunE(command, []string{})
	require.NoError(t, err)

	// Verify DB was created and contains progress
	dbPath := filepath.Join(tempDir, internal.AddressBalanceSequentialCheck+".db")
	assert.FileExists(t, dbPath, "Database file should exist")

	// Open DB to check results
	db, err := api.NewDB(tempDir, internal.AddressBalanceSequentialCheck)
	require.NoError(t, err)
	defer db.Close()

	assertResults(t, db)
}
