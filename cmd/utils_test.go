package cmd

import (
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/stretchr/testify/require"
	typesV1 "github.com/zondax/fil-parser/parser/v1/types"
)

func Test_filterSubcallsV1(t *testing.T) {
	addr1, err := address.NewFromString("f01234")
	require.NoError(t, err)
	addr2, err := address.NewFromString("f05678")
	require.NoError(t, err)

	addr3, err := address.NewFromString("f05679")
	require.NoError(t, err)

	data := typesV1.ExecutionTraceV1{
		Msg: &types.Message{
			To:   addr1,
			From: addr2,
		},
		Subcalls: []typesV1.ExecutionTraceV1{
			{
				Msg: &types.Message{
					To:   addr1,
					From: addr2,
				},
			},
			{
				Msg: &types.Message{
					To:   addr1,
					From: addr2,
				},
				Subcalls: []typesV1.ExecutionTraceV1{
					{
						Msg: &types.Message{
							To:   addr1,
							From: addr3,
						},
						Subcalls: []typesV1.ExecutionTraceV1{
							{
								Msg: &types.Message{
									To:   addr1,
									From: addr3,
								},
							},
						},
					},
				},
			},
		},
	}
	got := filterSubcallsV1(map[string]bool{"f05679": true}, data.Subcalls)
	assertSubCallsV1(t, map[string]bool{"f05679": true}, got)
}

func Test_filterSubcallsV2(t *testing.T) {
	addr1, err := address.NewFromString("f01234")
	require.NoError(t, err)
	addr2, err := address.NewFromString("f05678")
	require.NoError(t, err)

	addr3, err := address.NewFromString("f05679")
	require.NoError(t, err)

	data := types.ExecutionTrace{
		Msg: types.MessageTrace{
			To:   addr1,
			From: addr2,
		},
		Subcalls: []types.ExecutionTrace{
			{
				Msg: types.MessageTrace{
					To:   addr1,
					From: addr2,
				},
			},
			{
				Msg: types.MessageTrace{
					To:   addr1,
					From: addr2,
				},
				Subcalls: []types.ExecutionTrace{
					{
						Msg: types.MessageTrace{
							To:   addr1,
							From: addr3,
						},
						Subcalls: []types.ExecutionTrace{
							{
								Msg: types.MessageTrace{
									To:   addr1,
									From: addr3,
								},
							},
						},
					},
				},
			},
		},
	}
	got := filterSubcallsV2(map[string]bool{"f05679": true}, data.Subcalls)
	assertSubCallsV2(t, map[string]bool{"f05679": true}, got)
}

func assertSubCallsV1(t *testing.T, addrs map[string]bool, got []typesV1.ExecutionTraceV1) {
	for _, subcall := range got {
		require.Truef(t, (addrs[subcall.Msg.To.String()] || addrs[subcall.Msg.From.String()]), "all subcalls should contain filtered address")
		assertSubCallsV1(t, addrs, subcall.Subcalls)
	}
}

func assertSubCallsV2(t *testing.T, addrs map[string]bool, got []types.ExecutionTrace) {
	for _, subcall := range got {
		require.Truef(t, (addrs[subcall.Msg.To.String()] || addrs[subcall.Msg.From.String()]), "all subcalls should contain filtered address")
		assertSubCallsV2(t, addrs, subcall.Subcalls)
	}
}
