//go:build integration
// +build integration

package tests

import (
	"embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal/types"
)

//go:embed test_data.json
var testDataFs embed.FS

// testData is loaded once at init time
var testData TestData

type TestData struct {
	ValidateJSON struct {
		Start uint64 `json:"start"`
		End   uint64 `json:"end"`
	} `json:"validate-json"`
	ValidateNullBlocks struct {
		Start uint64 `json:"start"`
		End   uint64 `json:"end"`
	} `json:"validate-null-blocks"`
	ValidateCanonicalChain struct {
		Start uint64 `json:"start"`
		End   uint64 `json:"end"`
	} `json:"validate-canonical-chain"`
	ValidateAddressBalance struct {
		Addresses []string `json:"addresses"`
	} `json:"validate-address-balance"`
	ValidateMultisigState struct {
		Addresses []string `json:"addresses"`
	} `json:"validate-multisig-state"`
	ValidateAddressBalanceSequential struct {
		Addresses []string `json:"addresses"`
		Start     int64    `json:"start"`
		End       int64    `json:"end"`
	} `json:"validate-address-balance-sequential"`
	ValidateMultisigStateSequential struct {
		Addresses []string `json:"addresses"`
		Start     int64    `json:"start"`
		End       int64    `json:"end"`
	} `json:"validate-multisig-state-sequential"`
}

func init() {
	tmp, err := loadTestData()
	if err != nil {
		panic(err)
	}
	testData = tmp
}

func loadTestData() (TestData, error) {
	data := TestData{}
	contents, err := testDataFs.ReadFile("test_data.json")
	if err != nil {
		return data, err
	}

	if err := json.Unmarshal(contents, &data); err != nil {
		return data, err
	}
	return data, nil
}

func assertResults(t *testing.T, db *api.DB) {
	data, err := db.GetAllKVAsJSON()
	require.NoError(t, err)

	progress := map[string]types.Progress{}
	require.NoError(t, json.Unmarshal(data, &progress))

	for k, v := range progress {
		assert.True(t, v.Success, "Check failed for %s: %s", k, v.Message)
	}
}
