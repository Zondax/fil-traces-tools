package cmd

import (
	"errors"

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

func ValidateMultisigStateSequentialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   internal.MultisigStateSequentialCheck,
		Short: "Validate Multisig State Sequentially from start=1 (unless defined) to end",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return validateMultisigStateSequential(cmd)
		},
	}

	cmd.Flags().String(internal.AddressFileFlag, "", "path to a newline separated address file for addresses to check state")
	cmd.Flags().String(internal.DBPathFlag, ".", "path to a db file")
	cmd.Flags().Int64(internal.StartFlag, 1, "optional start height to validate")
	cmd.Flags().Int64(internal.EndFlag, 100, "end height")
	return cmd
}

type MsigAddress struct {
	Address             string
	Actor               *filTypes.Actor
	ParsedAddress       address.Address
	State               *types.MultisigState
	EquivalentAddresses map[string]bool
}

func validateMultisigStateSequential(cmd *cobra.Command) error {
	config := api.GetGlobalConfigs()
	log := initLogger()
	ctx := cmd.Context()

	dbPath, err := cmd.Flags().GetString(internal.DBPathFlag)
	if err != nil {
		log.Error("failed to get db path", zap.Error(err))
		return err
	}
	db, err := api.NewDB(dbPath, internal.MultisigStateSequentialCheck)
	if err != nil {
		log.Error("failed to create db", zap.Error(err), zap.String("db-path", dbPath))
		return err
	}
	stateDB, err := api.NewDB(dbPath, internal.MultisigStateSequentialCheck+".state")
	if err != nil {
		log.Error("failed to create state db", zap.Error(err))
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
		log.Error("failed to create db", zap.Error(err), zap.String("db-path", dbPath))
		return err
	}
	addresses, err := internal.ReadAddressFile(addressFile)
	if err != nil {
		log.Error("failed to read address file", zap.Error(err), zap.String("address-file", addressFile))
		return err
	}
	start := int64(1)
	if cmd.Flags().Changed(internal.StartFlag) {
		startHeight, err := cmd.Flags().GetInt64(internal.StartFlag)
		if err != nil {
			log.Error("could not get start height", zap.Error(err), zap.Int64("start-height", startHeight))
			return err
		}
		start = startHeight
	}
	endHeight, err := cmd.Flags().GetInt64(internal.EndFlag)
	if err != nil {
		log.Error("failed to get end height", zap.Error(err), zap.Int64("end-height", endHeight))
		return err
	}

	if endHeight < start {
		log.Error("end height is less than start height", zap.Int64("start-height", start), zap.Int64("end-height", endHeight))
		return errors.New("end height is less than start height")
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

	latestHeight, err := db.GetLatestHeight()
	if err != nil {
		log.Error("failed to get latest height", zap.Error(err))
		return err
	}

	addressMap := map[string]*MsigAddress{}
	// all addresses equivalent addresses used for filtering the traces
	allEquivalentAddresses := map[string]bool{}
	for _, addr := range addresses {
		parsedAddress, err := address.NewFromString(addr)
		if err != nil {
			log.Error("failed to parse provided address", zap.Error(err), zap.String("address", addr))
			internal.UpdateProgressAddress(addr, 0, false, err.Error(), db)
			return err
		}
		equivalentAddresses, err := internal.GetEquivalentAddresses(ctx, parsedAddress, rpcClient.FullNodeClient())
		if err != nil {
			log.Error("failed to get equivalent addresses", zap.Error(err), zap.String("address", addr))
			return err
		}
		for equivalentAddress := range equivalentAddresses {
			allEquivalentAddresses[equivalentAddress] = true
		}
		actor, err := rpcClient.FullNodeClient().StateGetActor(ctx, parsedAddress, filTypes.EmptyTSK)
		if err != nil {
			log.Error("failed to get onchain actor", zap.Error(err), zap.String("address", addr))
			return err
		}

		state := &types.MultisigState{}
		err = internal.GetProgressAddressState(addr, state, stateDB)
		if err != nil {
			log.Error("failed to get last state", zap.Error(err), zap.String("address", addr))
			return err
		}
		if state.Height > 0 && state.Height != latestHeight {
			log.Error("state height is not equal to latest height", zap.Int64("state-height", state.Height), zap.Int64("latest-height", latestHeight), zap.String("address", addr))
			state = &types.MultisigState{}
			latestHeight = start
		}
		address := MsigAddress{
			Address:             addr,
			Actor:               actor,
			ParsedAddress:       parsedAddress,
			State:               state,
			EquivalentAddresses: equivalentAddresses,
		}
		addressMap[addr] = &address
	}

	if latestHeight < endHeight && latestHeight > start {
		log.Info("resuming from latest height", zap.Int64("latest-height", latestHeight))
		start = latestHeight + 1
	}

	for height := start; height <= endHeight; height++ {
		log.Info("processing height", zap.Int64("height", height))
		data, err := api.GetTraceFromDataStore(height, dataStore, &config)
		if err != nil {
			log.Error("failed to get trace", zap.Error(err), zap.Int64("height", height))
			internal.UpdateProgressHeight(height, false, err.Error(), db)
			continue
		}
		data, err = filterTrace(height, allEquivalentAddresses, data)
		if err != nil {
			log.Error("failed to filter trace", zap.Error(err), zap.Int64("height", height))
			internal.UpdateProgressHeight(height, false, err.Error(), db)
			continue
		}
		tipset, err := api.ChainGetTipSetByHeight(ctx, height, rpcClient)
		if err != nil {
			log.Error("failed to get onchain tipset", zap.Error(err), zap.Int64("height", height))
			internal.UpdateProgressHeight(height, false, err.Error(), db)
			continue
		}
		// on-chain state is applied on the next tipset
		nextTipset, err := api.ChainGetTipSetByHeight(ctx, height+1, rpcClient)
		if err != nil {
			log.Error("failed to get next onchain tipset", zap.Error(err), zap.Int64("height", height))
			internal.UpdateProgressHeight(height, false, err.Error(), db)
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
			internal.UpdateProgressHeight(height, false, err.Error(), db)
			continue
		}
		if len(parsedTxData.Txs) == 0 {
			continue
		}
		msigEvents, err := parser.ParseMultisigEvents(ctx, parsedTxData.Txs, parsedTxData.Txs[0].TipsetCid, tipset.Key())
		if err != nil {
			log.Error("failed to parse multisig events", zap.Error(err), zap.Int64("height", height))
			internal.UpdateProgressHeight(height, false, err.Error(), db)
			continue
		}
		for _, addr := range addresses {
			log.Info("processing address", zap.String("address", addr), zap.Int64("height", height))
			if err := compareMultisigAddress(ctx, height, addressMap[addr], msigEvents, nextTipset, rpcClient); err != nil {
				log.Error("multisig state check failed", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
				internal.UpdateProgressAddress(addr, height, false, err.Error(), db)
			} else {
				internal.UpdateProgressAddress(addr, height, true, internal.ProgressOK, db)
			}
			if err := internal.UpdateProgressAddressState(addr, addressMap[addr].State, stateDB); err != nil {
				log.Error("failed to update state", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
			}
		}
		internal.UpdateProgressHeight(height, true, internal.ProgressOK, db)
	}
	return nil
}
