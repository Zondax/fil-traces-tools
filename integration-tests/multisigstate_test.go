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

// TestValidateMultisigStateCmd_Integration tests the actual command execution with S3 and RPC
// Run with: go test -tags=integration -v -run TestValidateMultisigStateCmd_Integration
func TestValidateMultisigStateCmd_Integration(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test DB
	tempDir, err := os.MkdirTemp("", "multisigstate_cmd_integration")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test address file with multisig addresses
	addressFile := filepath.Join(tempDir, "test_multisig_addresses.txt")
	testAddressesBytes := []byte(strings.Join(testData.ValidateMultisigState.Addresses, "\n")) // Using smaller actor IDs which are more likely to be multisigs
	err = os.WriteFile(addressFile, testAddressesBytes, 0644)
	require.NoError(t, err)

	// Create the command
	command := cmd.ValidateMultisigStateCmd()
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
	t.Logf("Running MultisigState validation with address file: %s", addressFile)
	err = command.RunE(command, []string{})
	require.NoError(t, err)

	// Verify DB was created and contains progress
	dbPath := filepath.Join(tempDir, internal.MultisigStateCheck+".db")
	assert.FileExists(t, dbPath, "Database file should exist")

	// Open DB to check results
	db, err := api.NewDB(tempDir, internal.MultisigStateCheck)
	require.NoError(t, err)
	defer db.Close()

	assertResults(t, db)
}

// TestValidateMultisigStateSequentialCmd_Integration tests the actual command execution with S3 and RPC
// Run with: go test -tags=integration -v -run TestValidateMultisigStateSequentialCmd_Integration
func TestValidateMultisigStateSequentialCmd_Integration(t *testing.T) {
	t.Parallel()

	// Create temporary directory for test DB
	tempDir, err := os.MkdirTemp("", "multisigstate_sequential_cmd_integration")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test address file with multisig addresses
	addressFile := filepath.Join(tempDir, "test_multisig_addresses.txt")
	testAddressesBytes := []byte(strings.Join(testData.ValidateMultisigState.Addresses, "\n")) // Using smaller actor IDs which are more likely to be multisigs
	err = os.WriteFile(addressFile, testAddressesBytes, 0644)
	require.NoError(t, err)

	// Create the command
	command := cmd.ValidateMultisigStateSequentialCmd()
	command.SetContext(t.Context())

	// Set test flags
	err = command.Flags().Set(internal.AddressFileFlag, addressFile)
	require.NoError(t, err)

	err = command.Flags().Set(internal.DBPathFlag, tempDir)
	require.NoError(t, err)

	err = command.Flags().Set(internal.StartFlag, fmt.Sprint(testData.ValidateMultisigStateSequential.Start))
	require.NoError(t, err)

	err = command.Flags().Set(internal.EndFlag, fmt.Sprint(testData.ValidateMultisigStateSequential.End))
	require.NoError(t, err)

	// Run the command
	t.Logf("Running MultisigState validation with address file: %s", addressFile)
	err = command.RunE(command, []string{})
	require.NoError(t, err)

	// Verify DB was created and contains progress
	dbPath := filepath.Join(tempDir, internal.MultisigStateSequentialCheck+".db")
	assert.FileExists(t, dbPath, "Database file should exist")

	// Open DB to check results
	db, err := api.NewDB(tempDir, internal.MultisigStateSequentialCheck)
	require.NoError(t, err)
	defer db.Close()

	assertResults(t, db)
}
