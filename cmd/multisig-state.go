package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	address "github.com/filecoin-project/go-address"
	filTypes "github.com/filecoin-project/lotus/chain/types"
	"github.com/spf13/cobra"
	fil_parser "github.com/zondax/fil-parser"
	"github.com/zondax/fil-parser/parser"
	parserTypes "github.com/zondax/fil-parser/types"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal"
	types "github.com/zondax/fil-trace-check/internal/types"
	"go.uber.org/zap"
)

func ValidateMultisigStateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   internal.MultisigStateCheck,
		Short: "Validate Multisig State",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return validateMultisigState(cmd)
		},
	}

	cmd.Flags().String(internal.AddressFileFlag, "", "path to a newline separated address file for addresses to check state")
	cmd.Flags().String(internal.DBPathFlag, ".", "path to a db file")
	cmd.Flags().String(internal.EventProviderFlag, "beryx", "event provider to use")
	cmd.Flags().String(internal.EventProviderTokenFlag, "", "event provider token")
	return cmd
}

func validateMultisigState(cmd *cobra.Command) error {
	config := api.GetGlobalConfigs()
	log := initLogger()
	ctx := cmd.Context()

	dbPath, err := cmd.Flags().GetString(internal.DBPathFlag)
	if err != nil {
		log.Error("failed to get db path", zap.Error(err))
		return err
	}
	db, err := api.NewDB(dbPath, internal.MultisigStateCheck)
	if err != nil {
		log.Error("failed to create db", zap.Error(err), zap.String("db-path", dbPath))
		return err
	}
	stateDB, err := api.NewDB(dbPath, internal.MultisigStateCheck+".state")
	if err != nil {
		log.Error("failed to create state db", zap.Error(err), zap.String("db-path", dbPath))
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close db", zap.Error(err))
		}
		if err := stateDB.Close(); err != nil {
			log.Error("failed to close state db", zap.Error(err))
		}
	}()

	addressFile, err := cmd.Flags().GetString(internal.AddressFileFlag)
	if err != nil {
		log.Error("failed to create db", zap.Error(err), zap.String("db-path", dbPath))
		return err
	}
	eventProvideName, err := cmd.Flags().GetString(internal.EventProviderFlag)
	if err != nil {
		log.Error("failed to get event provider", zap.Error(err))
		return err
	}
	eventProviderToken, err := cmd.Flags().GetString(internal.EventProviderTokenFlag)
	if err != nil {
		log.Error("failed to get event provider token", zap.Error(err))
		return err
	}
	eventProvider, err := types.NewEventProvider(eventProvideName, eventProviderToken)
	if err != nil {
		log.Error("failed to create event provider", zap.Error(err))
		return err
	}
	addresses, err := internal.ReadAddressFile(addressFile)
	if err != nil {
		log.Error("failed to read address file", zap.Error(err), zap.String("address-file", addressFile))
		return err
	}

	rpcClient, err := api.NewFilecoinRPCClient(ctx, config.NodeURL, config.NodeToken)
	if err != nil {
		log.Error("failed to get rpc client", zap.Error(err))
		return err
	}
	dataStore, err := api.GetDataStoreClient(&config)
	if err != nil {
		log.Error("failed to get data store client", zap.Error(err))
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
		log.Debug(fmt.Sprintf("Validating multisig state for address %s", addr))
		parsedAddr, err := address.NewFromString(addr)
		if err != nil {
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			continue
		}

		actor, err := rpcClient.FullNodeClient().StateGetActor(ctx, parsedAddr, filTypes.EmptyTSK)
		if err != nil {
			log.Error("failed to get onchain actor", zap.Error(err), zap.String("address", addr))
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			continue
		}

		heights, err := eventProvider.GetAddressEventHeights(ctx, addr)
		if err != nil {
			log.Error("failed to get onchain address events", zap.Error(err), zap.String("address", addr))
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			continue
		}
		equivalentAddresses, err := internal.GetEquivalentAddresses(ctx, parsedAddr, rpcClient.FullNodeClient())
		if err != nil {
			log.Error("failed to get equivalent addresses", zap.Error(err), zap.String("address", addr))
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			continue
		}

		processedHeights := map[int64]bool{}
		// try load state
		state := &types.MultisigState{}
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

		msigAddress := &MsigAddress{
			Address:             addr,
			Actor:               actor,
			ParsedAddress:       parsedAddr,
			State:               state,
			EquivalentAddresses: equivalentAddresses,
		}
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

			data, err = filterTrace(height, msigAddress.EquivalentAddresses, data)
			if err != nil {
				log.Error("failed to filter trace", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}

			tipset, err := api.ChainGetTipSetByHeight(ctx, height, rpcClient)
			if err != nil {
				log.Error("failed to get onchain tipset", zap.Error(err), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}

			// on-chain state is applied on the next tipset
			nextTipset, err := api.ChainGetTipSetByHeight(ctx, height+1, rpcClient)
			if err != nil {
				log.Error("failed to get onchain tipset", zap.Error(err), zap.Int64("height", height))
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
				log.Error("failed to parse transactions", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}

			if len(parsedTxData.Txs) == 0 {
				continue
			}

			msigEvents, err := parser.ParseMultisigEvents(ctx, parsedTxData.Txs, parsedTxData.Txs[0].TipsetCid, tipset.Key())
			if err != nil {
				log.Error("failed to parse multisig events", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
				continue
			}
			if err := compareMultisigAddress(ctx, height, msigAddress, msigEvents, nextTipset, rpcClient); err != nil {
				log.Error("failed to compare multisig state", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
			} else {
				internal.UpdateProgressAddress(addr, height, true, internal.ProgressOK, db)
			}
			if err := internal.UpdateProgressAddressState(addr, msigAddress.State, stateDB); err != nil {
				log.Error("failed to update address state", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
			}
			lastHeight = height
		}

		internal.UpdateProgressAddress(addr, lastHeight, true, internal.ProgressOK, db)
	}
	return nil
}

func compareMultisigAddress(ctx context.Context, height int64, addr *MsigAddress, msigEvents *parserTypes.MultisigEvents, tipset *filTypes.TipSet, rpcClient api.RPCClientInterface) error {
	if err := applyMultisigStateFromEvents(ctx, height, addr.State, msigEvents.MultisigInfo, rpcClient); err != nil {
		return fmt.Errorf("failed to apply multisig state from events")

	}
	msigOnChainState, err := rpcClient.FullNodeClient().StateReadState(ctx, addr.ParsedAddress, tipset.Key())
	if err != nil {
		return fmt.Errorf("failed to read state: %s", err)
	}

	onChainState := msigOnChainState.State.(map[string]interface{})

	onChainUnlockDurationRaw, ok := onChainState["UnlockDuration"].(float64)
	if !ok {
		return fmt.Errorf("failed to get unlock duration")
	}
	onChainUnlockDuration := int64(onChainUnlockDurationRaw)

	onChainSigners, ok := onChainState["Signers"].([]any)
	if !ok {
		return fmt.Errorf("failed to get signers")
	}

	onChainLockedBalanceStr, ok := onChainState["InitialBalance"].(string)
	if !ok {
		return fmt.Errorf("failed to get locked balance")
	}
	onChainLockedBalance, ok := big.NewInt(0).SetString(onChainLockedBalanceStr, 10)
	if !ok {
		return fmt.Errorf("failed to paarse locked balance")
	}

	// check we have the same number of signers
	if len(addr.State.Signers) != len(onChainSigners) {
		return fmt.Errorf("multisig signers mismatch for %s at height: %d: onchain=%d, parsed=%d", addr.Address, tipset.Height(), len(onChainSigners), len(addr.State.Signers))
	}

	// check that the signers are the same ( including equivalent addresses )
	onChainSignerMap := map[string]bool{}
	for _, signer := range onChainSigners {
		// onChainSignerMap[signer.String()] = true
		onChainSignerMap[signer.(string)] = true
		signerAddr, err := address.NewFromString(signer.(string))
		if err != nil {
			return fmt.Errorf("failed to parse signer address: %s : %w", signer.(string), err)
		}
		// get equivalent addresses for the signer
		equivalentAddresses, err := internal.GetEquivalentAddresses(ctx, signerAddr, rpcClient.FullNodeClient())
		if err != nil {
			return fmt.Errorf("failed to get equivalent addresses for signer: %s :%w", signerAddr.String(), err)

		}
		for equivalentAddress := range equivalentAddresses {
			onChainSignerMap[equivalentAddress] = true
		}
	}

	signerCheckFailed := false
	for _, signer := range addr.State.Signers {
		if _, ok := onChainSignerMap[signer]; !ok {
			signerCheckFailed = true
			break
		}
	}

	if signerCheckFailed {
		onChainSignerJson, _ := json.Marshal(onChainSignerMap)
		parsedSignerJson, _ := json.Marshal(addr.State.Signers)
		return fmt.Errorf("multisig signer mismatch for %s at height: %d: onchain=%s, parsed=%s", addr.Address, tipset.Height(), string(onChainSignerJson), string(parsedSignerJson))
	}

	if addr.State.LockedBalance == "" {
		addr.State.LockedBalance = big.NewInt(0).String()
	}
	if addr.State.LockedBalance != onChainLockedBalance.String() {
		return fmt.Errorf("multisig locked balance mismatch for %s at height: %d: onchain=%s, parsed=%s", addr.Address, tipset.Height(), onChainLockedBalance.String(), addr.State.LockedBalance)
	}

	if addr.State.UnlockDuration != onChainUnlockDuration {
		return fmt.Errorf("multisig unlock duration mismatch for %s at height: %d: onchain=%d, parsed=%d", addr.Address, tipset.Height(), onChainUnlockDuration, addr.State.UnlockDuration)
	}

	return nil
}

func applyMultisigStateFromEvents(ctx context.Context, height int64, msigState *types.MultisigState, msigEvents []*parserTypes.MultisigInfo, rpcClient api.RPCClientInterface) error {
	for _, msigEvent := range msigEvents {
		switch msigEvent.ActionType {
		case parser.MethodConstructor:
			constructor := types.Constructor{}
			if err := json.Unmarshal([]byte(msigEvent.Value), &constructor); err != nil {
				return fmt.Errorf("failed to parse constructor(%s): %s", msigEvent.Value, err)
			}
			msigState.Signers = constructor.Signers
			msigState.LockedBalance = constructor.LockedBalance
			msigState.UnlockDuration = constructor.UnlockDuration
		case parser.MethodAddSigner:
			addSigner := types.AddSigner{}
			if err := json.Unmarshal([]byte(msigEvent.Value), &addSigner); err != nil {
				return fmt.Errorf("failed to parse addSigner(%s): %s", msigEvent.Value, err)
			}
			msigState.Signers = append(msigState.Signers, addSigner.Signer)
		case parser.MethodSwapSigner:
			swapSigner := types.SwapSigner{}
			if err := json.Unmarshal([]byte(msigEvent.Value), &swapSigner); err != nil {
				return fmt.Errorf("failed to parse swapSigner(%s): %s", msigEvent.Value, err)
			}
			addr, err := address.NewFromString(swapSigner.From)
			if err != nil {
				return fmt.Errorf("failed to parse swapSigner.From(%s): %s", swapSigner.From, err)
			}
			// get equivalent signer for swapSigner.From
			equivalentSignerFrom, err := internal.GetEquivalentAddresses(ctx, addr, rpcClient.FullNodeClient())
			if err != nil {
				return fmt.Errorf("failed to get equivalent swapsigner.From(%s): %s", swapSigner.From, err)
			}
			newSigners := []string{swapSigner.To}
			for _, signer := range msigState.Signers {
				if equivalentSignerFrom[signer] {
					continue
				}
				newSigners = append(newSigners, signer)
			}
			msigState.Signers = newSigners

		case parser.MethodRemoveSigner:
			removeSigner := types.RemoveSigner{}
			if err := json.Unmarshal([]byte(msigEvent.Value), &removeSigner); err != nil {
				return fmt.Errorf("failed to parse removeSigner(%s): %s", msigEvent.Value, err)
			}
			addr, err := address.NewFromString(removeSigner.Signer)
			if err != nil {
				return fmt.Errorf("failed to parse removeSigner.Signer(%s): %s", removeSigner.Signer, err)
			}
			// get equivalent signer for removeSigner.Signer
			equivalentSignerRemove, err := internal.GetEquivalentAddresses(ctx, addr, rpcClient.FullNodeClient())
			if err != nil {
				return fmt.Errorf("failed to get equivalent removeSigner.Signer(%s): %s", removeSigner.Signer, err)
			}

			newSigners := []string{}
			for _, signer := range msigState.Signers {
				if equivalentSignerRemove[signer] {
					continue
				}
				newSigners = append(newSigners, signer)
			}
			msigState.Signers = newSigners
		case parser.MethodLockBalance:
			lockBalance := types.LockBalance{}
			if err := json.Unmarshal([]byte(msigEvent.Value), &lockBalance); err != nil {
				return fmt.Errorf("failed to parse lockBalance(%s): %s", msigEvent.Value, err)
			}
			msigState.LockedBalance = lockBalance.Amount
			msigState.UnlockDuration = lockBalance.UnlockDuration
		}
	}
	msigState.Height = height
	return nil
}
