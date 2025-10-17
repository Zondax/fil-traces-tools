package cmd

import (
	"context"
	"fmt"
	"math/big"

	address "github.com/filecoin-project/go-address"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/spf13/cobra"
	fil_parser "github.com/zondax/fil-parser"
	parserTypes "github.com/zondax/fil-parser/types"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal"
	types "github.com/zondax/fil-trace-check/internal/types"
	"go.uber.org/zap"
)

func ValidateAddressBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   internal.AddressBalanceCheck,
		Short: "Validate Address Balance",
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateAddressBalance(cmd)
		},
	}
	cmd.Flags().String(internal.AddressFileFlag, "", "path to a newline separated address file for addresses to check state")
	cmd.Flags().String(internal.DBPathFlag, ".", "path to the database")
	cmd.Flags().String(internal.EventProviderFlag, "beryx", "event provider to use")
	cmd.Flags().String(internal.EventProviderTokenFlag, "", "event provider token")
	return cmd
}

func validateAddressBalance(cmd *cobra.Command) error {
	config := api.GetGlobalConfigs()
	log := initLogger()
	ctx := cmd.Context()

	dbPath, err := cmd.Flags().GetString(internal.DBPathFlag)
	if err != nil {
		log.Error("could not get db path", zap.Error(err))
		return err
	}
	db, err := api.NewDB(dbPath, internal.AddressBalanceCheck)
	if err != nil {
		log.Error("could not create db", zap.Error(err))
		return err
	}
	stateDB, err := api.NewDB(dbPath, internal.AddressBalanceCheck+".state")
	if err != nil {
		log.Error("could not create state db", zap.Error(err))
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close database", zap.Error(err))
		}
		if err := stateDB.Close(); err != nil {
			log.Error("failed to close state database", zap.Error(err))
		}
	}()

	addressFile, err := cmd.Flags().GetString(internal.AddressFileFlag)
	if err != nil {
		log.Error("could not get address file", zap.Error(err), zap.String("address-file", addressFile))
		return err
	}
	eventProviderName, err := cmd.Flags().GetString(internal.EventProviderFlag)
	if err != nil {
		log.Error("could not get event provider", zap.Error(err), zap.String("event-provider", eventProviderName))
		return err
	}
	eventProviderToken, err := cmd.Flags().GetString(internal.EventProviderTokenFlag)
	if err != nil {
		log.Error("could not get event provider token", zap.Error(err), zap.String("event-provider-token", eventProviderToken))
		return err
	}
	eventProvider, err := types.NewEventProvider(eventProviderName, eventProviderToken)
	if err != nil {
		log.Error("could not create event provider", zap.Error(err), zap.String("event-provider", eventProviderName))
		return err
	}

	addresses, err := internal.ReadAddressFile(addressFile)
	if err != nil {
		log.Error("could not read address file", zap.Error(err), zap.String("address-file", addressFile))
		return err
	}
	rpcClient, err := api.NewFilecoinRPCClient(ctx, config.NodeURL, config.NodeToken)
	if err != nil {
		log.Error("could not create rpc client", zap.Error(err), zap.String("node-url", config.NodeURL))
		return err
	}
	dataStore, err := api.GetDataStoreClient(&config)
	if err != nil {
		log.Error("could not create data store client", zap.Error(err))
		return err
	}

	parser, err := fil_parser.NewFilecoinParserWithActorV2(
		rpcClient.RosettaLib(), api.GetDataSource(&config, rpcClient),
		getParserLogger(),
	)

	if err != nil {
		log.Error("failed to create parser", zap.Error(err))
		return err
	}

	for _, addr := range addresses {
		log.Debug(fmt.Sprintf("Validating address balance for %s", addr))
		parsedAddress, err := address.NewFromString(addr)
		if err != nil {
			log.Error("failed to parse provided address", zap.Error(err), zap.String("address", addr))
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			continue
		}
		equivalentAddresses, err := internal.GetEquivalentAddresses(ctx, parsedAddress, rpcClient.FullNodeClient())
		if err != nil {
			log.Error("failed to get equivalent addresses", zap.Error(err), zap.String("address", addr))
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			continue
		}
		heights, err := eventProvider.GetAddressEventHeights(ctx, addr)
		if err != nil {
			log.Error("failed to get address events", zap.Error(err), zap.String("address", addr))
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			continue
		}
		processedHeights := map[int64]bool{}

		// try load state
		state := &types.AddressState{}
		err = internal.GetProgressAddressState(addr, state, stateDB)
		if err != nil {
			log.Error("failed to get last state", zap.Error(err), zap.String("address", addr))
		}
		if state.Height > 0 {
			for _, height := range heights {
				if height <= state.Height {
					processedHeights[height] = true
				}
			}
		}

		addrInfo := &Address{
			ParsedAddress:       parsedAddress,
			EquivalentAddresses: equivalentAddresses,
			State:               state,
		}
		log.Debug("got address events", zap.Int("count", len(heights)), zap.String("address", addr))

		lastHeight := int64(0)
		for _, height := range heights {
			if processedHeights[height] {
				continue
			}
			processedHeights[height] = true
			data, err := api.GetTraceFromDataStore(height, dataStore, &config)
			if err != nil {
				log.Error("failed to get trace", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}
			data, err = filterTrace(height, equivalentAddresses, data)
			if err != nil {
				log.Error("failed to filter trace", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}
			tipset, err := api.ChainGetTipSetByHeight(ctx, height, rpcClient)
			if err != nil {
				log.Error("failed to get tipset", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}
			nextTipset, err := api.ChainGetTipSetByHeight(ctx, height+1, rpcClient)
			if err != nil {
				log.Error("failed to get next tipset", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}

			txsData := parserTypes.TxsData{
				Traces: data,
				Tipset: &parserTypes.ExtendedTipSet{
					TipSet: *tipset,
				},
			}
			nodeInfo := api.HeightToNodeVersion(height)
			txsData.Metadata.NodeInfo = *nodeInfo

			parsedTxData, err := parser.ParseTransactions(ctx, txsData)
			if err != nil {
				log.Error("failed to parse transactions", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}
			if len(parsedTxData.Txs) == 0 {
				continue
			}
			if err := compareAddressBalance(ctx, height, addrInfo, nextTipset, parsedTxData, rpcClient); err != nil {
				log.Error("failed to compare address balance", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
			} else {
				internal.UpdateProgressAddress(addr, height, true, internal.ProgressOK, db)
			}
			if err := internal.UpdateProgressAddressState(addr, addrInfo.State, stateDB); err != nil {
				log.Error("failed to update address state", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
			}
			lastHeight = height
		}
		internal.UpdateProgressHeight(lastHeight, true, internal.ProgressOK, db)
	}
	return nil
}

func compareAddressBalance(ctx context.Context, height int64, addr *Address, tipset *filTypes.TipSet, parsedTxData *parserTypes.TxsParsedResult, rpcClient api.RPCClientInterface) error {
	applyAddressBalanceStateFromTransactions(height, addr.EquivalentAddresses, addr.State, parsedTxData.Txs)

	sent := big.NewInt(0)
	if addr.State.Sent != nil {
		sent.Set(addr.State.Sent)
	}
	received := big.NewInt(0)
	if addr.State.Received != nil {
		received.Set(addr.State.Received)
	}
	parsedBalance := received.Sub(received, sent)
	// check that tokens were received before sending (non-negative balance)
	if parsedBalance.Cmp(big.NewInt(0)) < 0 {
		return fmt.Errorf("negative balance for %s", addr.ParsedAddress)
	}

	// check that onchain and parsed balance match
	actor, err := rpcClient.FullNodeClient().StateReadState(ctx, addr.ParsedAddress, tipset.Key())
	if err != nil {
		return fmt.Errorf("failed to get onchain address balance: %w", err)
	}

	if actor.Balance.Cmp(parsedBalance) != 0 {
		return fmt.Errorf("balance mismatch for %s: onchain=%s, parsed=%s", addr.ParsedAddress, actor.Balance.String(), parsedBalance.String())
	}
	return nil
}

func applyAddressBalanceStateFromTransactions(height int64, equivalentAddresses map[string]bool, addressState *types.AddressState, txs []*parserTypes.Transaction) {
	for _, tx := range txs {
		if tx.Status != "Ok" {
			continue
		}
		if equivalentAddresses[tx.TxTo] || equivalentAddresses[tx.TxFrom] {
			// tx to will be the recipeint
			addressState.Height = height
			if equivalentAddresses[tx.TxTo] && tx.Amount != nil {
				if addressState.Received != nil {
					addressState.Received = addressState.Received.Add(addressState.Received, tx.Amount)
				} else {
					addressState.Received = new(big.Int).Set(tx.Amount)
				}
			}
			if equivalentAddresses[tx.TxFrom] {
				total := big.NewInt(0)
				if tx.Amount != nil {
					total = total.Add(total, tx.Amount)
				}
				if addressState.Sent != nil {
					addressState.Sent = addressState.Sent.Add(addressState.Sent, total)
				} else {
					addressState.Sent = new(big.Int).Set(total)
				}
			}
		}
	}

}
