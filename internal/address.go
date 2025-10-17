package internal

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	filTypes "github.com/filecoin-project/lotus/chain/types"
)

func ReadAddressFile(addressFile string) ([]string, error) {
	// Clean the file path to prevent directory traversal attacks
	cleanPath := filepath.Clean(addressFile)
	addressFileBytes, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}
	addresses := []string{}
	for _, line := range strings.Split(string(addressFileBytes), "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			addresses = append(addresses, trimmed)
		}
	}
	return addresses, nil
}

func GetEquivalentAddresses(ctx context.Context, add address.Address, rpcClient api.FullNode) (map[string]bool, error) {
	addresses := map[string]bool{
		add.String(): true,
	}
	actor, err := rpcClient.StateGetActor(ctx, add, filTypes.EmptyTSK)
	if err != nil {
		return nil, err
	}
	if actor.DelegatedAddress != nil {
		addresses[actor.DelegatedAddress.String()] = true
	}
	if isRobustAddress(add) {
		// get id address
		idAddress, err := rpcClient.StateLookupID(ctx, add, filTypes.EmptyTSK)
		if err != nil {
			return nil, err
		}
		addresses[idAddress.String()] = true
		return addresses, nil
	}
	key, err := rpcClient.StateAccountKey(ctx, add, filTypes.EmptyTSK)
	if err != nil {
		if strings.Contains(err.Error(), "actor code is not account") {
			robustAddress, err := rpcClient.StateLookupRobustAddress(ctx, add, filTypes.EmptyTSK)
			if err != nil {
				return nil, err
			}
			addresses[robustAddress.String()] = true
		} else {
			return addresses, err
		}
	} else {
		addresses[key.String()] = true
	}
	return addresses, nil
}

func isRobustAddress(add address.Address) bool {
	switch add.Protocol() {
	case address.BLS, address.SECP256K1, address.Actor, address.Delegated:
		return true
	case address.ID:
		return false
	default:
		// Consider unknown type as robust
		return true
	}
}
