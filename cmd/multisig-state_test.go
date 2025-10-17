package cmd

import (
	"encoding/json"
	"testing"

	address "github.com/filecoin-project/go-address"
	lotusAPI "github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	parserTypes "github.com/zondax/fil-parser/types"
	"github.com/zondax/fil-trace-check/internal/mocks"
	types "github.com/zondax/fil-trace-check/internal/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
)

type MockRPCClient struct {
	client lotusAPI.FullNode
}

func (m *MockRPCClient) FullNodeClient() lotusAPI.FullNode {
	return m.client
}
func (m *MockRPCClient) RosettaLib() *rosettaFilecoinLib.RosettaConstructionFilecoin {
	return nil
}
func (m *MockRPCClient) NodeInfo() parserTypes.NodeInfo {
	return parserTypes.NodeInfo{}
}

func TestApplyMultisigStateFromEvents_Constructor(t *testing.T) {
	state := &types.MultisigState{}

	constructor := types.Constructor{
		Signers:        []string{"f1234", "f5678", "f9012"},
		LockedBalance:  "1000000",
		UnlockDuration: 100,
	}
	constructorJSON, err := json.Marshal(constructor)
	require.NoError(t, err)

	events := []*parserTypes.MultisigInfo{
		{
			ActionType: "Constructor",
			Value:      string(constructorJSON),
		},
	}

	err = applyMultisigStateFromEvents(t.Context(), 0, state, events, nil)
	require.NoError(t, err)

	assert.Equal(t, constructor.Signers, state.Signers)
	assert.Equal(t, constructor.LockedBalance, state.LockedBalance)
	assert.Equal(t, constructor.UnlockDuration, state.UnlockDuration)
}

func TestApplyMultisigStateFromEvents_AddSigner(t *testing.T) {
	state := &types.MultisigState{
		Signers: []string{"f1234", "f5678"},
	}

	addSigner := types.AddSigner{
		Signer: "f9012",
	}
	addSignerJSON, err := json.Marshal(addSigner)
	require.NoError(t, err)

	events := []*parserTypes.MultisigInfo{
		{
			ActionType: "AddSigner",
			Value:      string(addSignerJSON),
		},
	}

	err = applyMultisigStateFromEvents(t.Context(), 0, state, events, nil)
	require.NoError(t, err)

	assert.Equal(t, []string{"f1234", "f5678", "f9012"}, state.Signers)
}

func TestApplyMultisigStateFromEvents_RemoveSigner(t *testing.T) {
	state := &types.MultisigState{
		Signers: []string{"f01234", "f05678", "f09012"},
	}

	removeSigner := types.RemoveSigner{
		Signer: "f05678",
	}
	removeSignerJSON, err := json.Marshal(removeSigner)
	require.NoError(t, err)

	events := []*parserTypes.MultisigInfo{
		{
			ActionType: "RemoveSigner",
			Value:      string(removeSignerJSON),
		},
	}

	fullNodeMock := mocks.NewFullNode(t)
	fullNodeMock.On("StateGetActor", mock.Anything, mock.Anything, mock.Anything).Return(&filTypes.Actor{}, nil)
	fullNodeMock.On("StateAccountKey", mock.Anything, mock.Anything, mock.Anything).Return(address.Address{}, nil)
	mockRPCClient := &MockRPCClient{
		client: fullNodeMock,
	}

	err = applyMultisigStateFromEvents(t.Context(), 0, state, events, mockRPCClient)
	require.NoError(t, err)

	assert.Equal(t, []string{"f01234", "f09012"}, state.Signers)
}

func TestApplyMultisigStateFromEvents_SwapSigner(t *testing.T) {
	state := &types.MultisigState{
		Signers: []string{"f01234", "f05678", "f09012"},
	}

	swapSigner := types.SwapSigner{
		From: "f05678",
		To:   "f03456",
	}
	swapSignerJSON, err := json.Marshal(swapSigner)
	require.NoError(t, err)

	events := []*parserTypes.MultisigInfo{
		{
			ActionType: "SwapSigner",
			Value:      string(swapSignerJSON),
		},
	}

	fullNodeMock := mocks.NewFullNode(t)
	fullNodeMock.On("StateGetActor", mock.Anything, mock.Anything, mock.Anything).Return(&filTypes.Actor{}, nil)
	fullNodeMock.On("StateAccountKey", mock.Anything, mock.Anything, mock.Anything).Return(address.Address{}, nil)
	mockRPCClient := &MockRPCClient{
		client: fullNodeMock,
	}

	err = applyMultisigStateFromEvents(t.Context(), 0, state, events, mockRPCClient)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"f01234", "f09012", "f03456"}, state.Signers)
}

func TestApplyMultisigStateFromEvents_LockBalance(t *testing.T) {
	state := &types.MultisigState{
		LockedBalance:  "500000",
		UnlockDuration: 50,
	}

	lockBalance := types.LockBalance{
		Amount:         "2000000",
		UnlockDuration: 200,
	}
	lockBalanceJSON, err := json.Marshal(lockBalance)
	require.NoError(t, err)

	events := []*parserTypes.MultisigInfo{
		{
			ActionType: "LockBalance",
			Value:      string(lockBalanceJSON),
		},
	}

	err = applyMultisigStateFromEvents(t.Context(), 0, state, events, nil)
	require.NoError(t, err)

	assert.Equal(t, lockBalance.Amount, state.LockedBalance)
	assert.Equal(t, lockBalance.UnlockDuration, state.UnlockDuration)
}

func TestApplyMultisigStateFromEvents_MultipleEvents(t *testing.T) {
	state := &types.MultisigState{}

	// Constructor
	constructor := types.Constructor{
		Signers:        []string{"f01234", "f05678"},
		LockedBalance:  "1000000",
		UnlockDuration: 100,
	}
	constructorJSON, _ := json.Marshal(constructor)

	// Add signer
	addSigner := types.AddSigner{
		Signer: "f09012",
	}
	addSignerJSON, _ := json.Marshal(addSigner)

	// Remove signer
	removeSigner := types.RemoveSigner{
		Signer: "f01234",
	}
	removeSignerJSON, _ := json.Marshal(removeSigner)

	// Lock balance
	lockBalance := types.LockBalance{
		Amount:         "2000000",
		UnlockDuration: 200,
	}
	lockBalanceJSON, _ := json.Marshal(lockBalance)

	events := []*parserTypes.MultisigInfo{
		{
			ActionType: "Constructor",
			Value:      string(constructorJSON),
		},
		{
			ActionType: "AddSigner",
			Value:      string(addSignerJSON),
		},
		{
			ActionType: "RemoveSigner",
			Value:      string(removeSignerJSON),
		},
		{
			ActionType: "LockBalance",
			Value:      string(lockBalanceJSON),
		},
	}

	fullNodeMock := mocks.NewFullNode(t)
	fullNodeMock.On("StateGetActor", mock.Anything, mock.Anything, mock.Anything).Return(&filTypes.Actor{}, nil)
	fullNodeMock.On("StateAccountKey", mock.Anything, mock.Anything, mock.Anything).Return(address.Address{}, nil)
	mockRPCClient := &MockRPCClient{
		client: fullNodeMock,
	}

	err := applyMultisigStateFromEvents(t.Context(), 0, state, events, mockRPCClient)
	require.NoError(t, err)

	// Final state should have signers: f05678, f09012
	assert.Equal(t, []string{"f05678", "f09012"}, state.Signers)
	assert.Equal(t, "2000000", state.LockedBalance)
	assert.Equal(t, int64(200), state.UnlockDuration)
}
