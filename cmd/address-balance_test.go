package cmd

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	parserTypes "github.com/zondax/fil-parser/types"
	types "github.com/zondax/fil-trace-check/internal/types"
)

func TestApplyAddressBalanceStateFromTransactions(t *testing.T) {
	tests := []struct {
		name         string
		addr         string
		initialState *types.AddressState
		txs          []*parserTypes.Transaction
		expectedSent *big.Int
		expectedRecv *big.Int
	}{
		{
			name:         "single transaction - receiving funds",
			addr:         "f1234",
			initialState: &types.AddressState{},
			txs: []*parserTypes.Transaction{
				{
					TxTo:   "f1234",
					TxFrom: "f5678",
					Amount: big.NewInt(1000),
					Status: "Ok",
				},
			},
			expectedSent: nil,
			expectedRecv: big.NewInt(1000),
		},
		{
			name:         "single transaction - sending funds",
			addr:         "f1234",
			initialState: &types.AddressState{},
			txs: []*parserTypes.Transaction{
				{
					TxTo:   "f5678",
					TxFrom: "f1234",
					Amount: big.NewInt(500),
					Status: "Ok",
				},
			},
			expectedSent: big.NewInt(500),
			expectedRecv: nil,
		},
		{
			name:         "multiple transactions - both sending and receiving",
			addr:         "f1234",
			initialState: &types.AddressState{},
			txs: []*parserTypes.Transaction{
				{
					TxTo:   "f1234",
					TxFrom: "f5678",
					Amount: big.NewInt(1000),
					Status: "Ok",
				},
				{
					TxTo:   "f9999",
					TxFrom: "f1234",
					Amount: big.NewInt(300),
					Status: "Ok",
				},
				{
					TxTo:   "f1234",
					TxFrom: "f8888",
					Amount: big.NewInt(500),
					Status: "Ok",
				},
			},
			expectedSent: big.NewInt(300),
			expectedRecv: big.NewInt(1500),
		},
		{
			name:         "transaction with zero amount - should not affect balance",
			addr:         "f1234",
			initialState: &types.AddressState{},
			txs: []*parserTypes.Transaction{
				{
					TxTo:   "f1234",
					TxFrom: "f5678",
					Amount: big.NewInt(0),
					Status: "Ok",
				},
			},
			expectedSent: nil,
			expectedRecv: big.NewInt(0),
		},
		{
			name:         "transaction with nil amount - should not affect balance",
			addr:         "f1234",
			initialState: &types.AddressState{},
			txs: []*parserTypes.Transaction{
				{
					TxTo:   "f1234",
					TxFrom: "f5678",
					Amount: nil,
					Status: "Ok",
				},
			},
			expectedSent: nil,
			expectedRecv: nil,
		},
		{
			name: "accumulate on existing state",
			addr: "f1234",
			initialState: &types.AddressState{
				Sent:     big.NewInt(200),
				Received: big.NewInt(800),
			},
			txs: []*parserTypes.Transaction{
				{
					TxTo:   "f1234",
					TxFrom: "f5678",
					Amount: big.NewInt(100),
					Status: "Ok",
				},
				{
					TxTo:   "f9999",
					TxFrom: "f1234",
					Amount: big.NewInt(50),
					Status: "Ok",
				},
			},
			expectedSent: big.NewInt(250),
			expectedRecv: big.NewInt(900),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &types.AddressState{}
			if tt.initialState.Sent != nil {
				state.Sent = new(big.Int).Set(tt.initialState.Sent)
			}
			if tt.initialState.Received != nil {
				state.Received = new(big.Int).Set(tt.initialState.Received)
			}

			applyAddressBalanceStateFromTransactions(0, map[string]bool{
				tt.addr: true,
			}, state, tt.txs)

			if tt.expectedSent == nil {
				assert.Nil(t, state.Sent)
			} else {
				require.NotNil(t, state.Sent)
				assert.Equal(t, 0, tt.expectedSent.Cmp(state.Sent))
			}

			if tt.expectedRecv == nil {
				assert.Nil(t, state.Received)
			} else {
				require.NotNil(t, state.Received)
				assert.Equal(t, 0, tt.expectedRecv.Cmp(state.Received))
			}
		})
	}
}
