package cmd

import (
	"errors"

	address "github.com/filecoin-project/go-address"
	"github.com/spf13/cobra"
	fil_parser "github.com/zondax/fil-parser"
	parserTypes "github.com/zondax/fil-parser/types"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal"
	types "github.com/zondax/fil-trace-check/internal/types"
	"go.uber.org/zap"
)

func ValidateAddressBalanceSequentialCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   internal.AddressBalanceSequentialCheck,
		Short: "Validate Address Balance Sequentially from start=1 (unless defined) to end",
		RunE: func(cmd *cobra.Command, args []string) error {
			return validateAddressBalanceSequential(cmd)
		},
	}
	cmd.Flags().String(internal.AddressFileFlag, "", "path to a newline separated address file for addresses to check state")
	cmd.Flags().String(internal.DBPathFlag, ".", "path to the database")
	cmd.Flags().Int64(internal.StartFlag, 1, "optional start height to validate")
	cmd.Flags().Int64(internal.EndFlag, 0, "end height to validate")
	return cmd
}

type Address struct {
	EquivalentAddresses map[string]bool
	State               *types.AddressState
	ParsedAddress       address.Address
}

func validateAddressBalanceSequential(cmd *cobra.Command) error {
	config := api.GetGlobalConfigs()
	log := initLogger()
	ctx := cmd.Context()

	dbPath, err := cmd.Flags().GetString(internal.DBPathFlag)
	if err != nil {
		log.Error("could not get db path", zap.Error(err))
		return err
	}

	db, err := api.NewDB(dbPath, internal.AddressBalanceSequentialCheck)
	if err != nil {
		log.Error("could not create db", zap.Error(err))
		return err
	}
	stateDB, err := api.NewDB(dbPath, internal.AddressBalanceSequentialCheck+".state")
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
	addresses, err := internal.ReadAddressFile(addressFile)
	if err != nil {
		log.Error("could not read address file", zap.Error(err), zap.String("address-file", addressFile))
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
		log.Error("could not get end height", zap.Error(err), zap.Int64("end-height", endHeight))
		return err
	}

	if endHeight < start {
		log.Error("end height is less than start height", zap.Int64("start-height", start), zap.Int64("end-height", endHeight))
		return errors.New("end height is less than start height")
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

	latestHeight, err := db.GetLatestHeight()
	if err != nil {
		log.Error("failed to get latest height", zap.Error(err))
		return err
	}

	addressMap := map[string]*Address{}
	// equivalent addresses for all addresses used to filter traces
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

		state := &types.AddressState{}
		err = internal.GetProgressAddressState(addr, state, stateDB)
		if err != nil {
			log.Error("failed to get last state", zap.Error(err), zap.String("address", addr))
			return err
		}

		if state.Height > 0 && state.Height != latestHeight {
			log.Info("address state not at latest height, resetting state and starting again", zap.String("address", addr), zap.Int64("state-height", state.Height), zap.Int64("start-height", start))
			state = &types.AddressState{}
			latestHeight = start
		}

		address := Address{
			EquivalentAddresses: equivalentAddresses,
			State:               state,
			ParsedAddress:       parsedAddress,
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
			log.Error("failed to get tipset", zap.Error(err), zap.Int64("height", height))
			internal.UpdateProgressHeight(height, false, err.Error(), db)
			continue
		}
		// on-chain state is applied on the next tipset
		nextTipset, err := api.ChainGetTipSetByHeight(ctx, height+1, rpcClient)
		if err != nil {
			log.Error("failed to get next tipset", zap.Error(err), zap.Int64("height", height))
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
		for _, addr := range addresses {
			log.Info("processing address", zap.String("address", addr), zap.Int64("height", height))
			if err := compareAddressBalance(ctx, height, addressMap[addr], nextTipset, parsedTxData, rpcClient); err != nil {
				log.Error("address balance check failed", zap.Error(err), zap.String("address", addr), zap.Int64("height", height))
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
