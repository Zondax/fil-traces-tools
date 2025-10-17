package api

import (
	"context"
	"fmt"
	"net/http"

	jsonrpc "github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/client"
	lotusChainTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/zondax/fil-parser/types"
	rosettaFilecoinLib "github.com/zondax/rosetta-filecoin-lib"
)

type RPCClientInterface interface {
	RosettaLib() *rosettaFilecoinLib.RosettaConstructionFilecoin
	NodeInfo() types.NodeInfo
	FullNodeClient() api.FullNode
}

type RPCClient struct {
	url          string
	token        string
	client       api.FullNode
	clientCloser *jsonrpc.ClientCloser
	ctx          context.Context
	rosettaLib   *rosettaFilecoinLib.RosettaConstructionFilecoin
	nodeInfo     types.NodeInfo
}

func (rpc *RPCClient) RosettaLib() *rosettaFilecoinLib.RosettaConstructionFilecoin {
	return rpc.rosettaLib
}

func (rpc *RPCClient) NodeInfo() types.NodeInfo {
	return rpc.nodeInfo
}

func (rpc *RPCClient) FullNodeClient() api.FullNode {
	return rpc.client
}

// NewFilecoinRPCClient creates a new blockchain RPC remoteNode
func NewFilecoinRPCClient(ctx context.Context, url string, token string) (RPCClientInterface, error) {

	headers := http.Header{}
	if len(token) > 0 {
		headers.Add("Authorization", "Bearer "+token)
	}

	lotusAPI, closer, err := client.NewFullNodeRPCV1(ctx, url, headers)

	if err != nil {
		return nil, err
	}

	// Setup rosetta lib
	r := rosettaFilecoinLib.NewRosettaConstructionFilecoin(lotusAPI)
	if r == nil {
		return nil, fmt.Errorf("could not create instance of rosetta filecoin-lib")
	}

	// Get node version
	nodeFullVersion, err := lotusAPI.Version(ctx)
	if err != nil {
		return nil, err
	}
	nodeInfo, err := processNodeVersion(nodeFullVersion.Version)
	if err != nil {
		return nil, err
	}

	return &RPCClient{
		url:          url,
		token:        token,
		client:       lotusAPI,
		clientCloser: &closer,
		ctx:          ctx,
		rosettaLib:   r,
		nodeInfo:     *nodeInfo,
	}, nil

}

func ChainGetTipSetByHeight(ctx context.Context, height int64, rpcClient RPCClientInterface) (*lotusChainTypes.TipSet, error) {
	tipset, err := rpcClient.FullNodeClient().ChainGetTipSetByHeight(ctx, abi.ChainEpoch(height), lotusChainTypes.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("could not get tipset file: %w", err)
	}
	return tipset, nil
}
