package cmd

import (
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/filecoin-project/go-state-types/abi"
	apitypes "github.com/filecoin-project/lotus/api"
	"github.com/spf13/cobra"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal"
	"go.uber.org/zap"
)

func ValidateNullBlocksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   internal.NullBlocksCheck,
		Short: "Validate Null Blocks",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return validateNullBlocks(cmd)
		},
	}
	cmd.Flags().Int64(internal.StartFlag, 1, "start height to validate")
	cmd.Flags().Int64(internal.EndFlag, 100, "end height to validate")
	cmd.Flags().String(internal.DBPathFlag, ".", "path to the database")
	return cmd
}

func validateNullBlocks(cmd *cobra.Command) error {
	config := api.GetGlobalConfigs()
	log := initLogger()
	ctx := cmd.Context()

	start, err := cmd.Flags().GetInt64(internal.StartFlag)
	if err != nil {
		log.Error("failed to get start", zap.Error(err))
		return err
	}
	end, err := cmd.Flags().GetInt64(internal.EndFlag)
	if err != nil {
		log.Error("failed to get end", zap.Error(err))
		return err
	}
	dbPath, err := cmd.Flags().GetString(internal.DBPathFlag)
	if err != nil {
		log.Error("failed to get db path", zap.Error(err))
		return err
	}
	db, err := api.NewDB(dbPath, internal.NullBlocksCheck)
	if err != nil {
		log.Error("failed to create db", zap.Error(err))
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close database", zap.Error(err))
		}
	}()

	rpcClient, err := api.NewFilecoinRPCClient(ctx, config.NodeURL, config.NodeToken)
	if err != nil {
		log.Error("failed to create rpc client", zap.Error(err))
		return err
	}
	dataStore, err := api.GetDataStoreClient(&config)
	if err != nil {
		log.Error("failed to create data store client", zap.Error(err))
		return err
	}
	latestHeight, err := db.GetLatestHeight()
	if err != nil {
		log.Error("failed to get latest height", zap.Error(err))
		return err
	}
	if latestHeight > 0 && latestHeight > start {
		log.Info("resuming from latest height", zap.Int64("latest-height", latestHeight))
		start = latestHeight
	}

	for i := start; i <= end; i++ {
		log.Debug(fmt.Sprintf("Validating null blocks for height %d", i))

		data, err := api.GetTraceFromDataStore(i, dataStore, &config)
		if err != nil {
			log.Error("failed to get trace", zap.Error(err), zap.Int64("height", i))
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		var computeState apitypes.ComputeStateOutput
		err = sonic.Unmarshal(data, &computeState)
		if err != nil {
			log.Error("failed to unmarshal trace", zap.Error(err), zap.Int64("height", i))
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		traceIsNull := len(computeState.Trace) == 0

		tipset, err := api.ChainGetTipSetByHeight(ctx, i, rpcClient)
		if err != nil {
			log.Error("failed to get onchain tipset", zap.Error(err), zap.Int64("height", i))
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		isNull := tipset.Height() != abi.ChainEpoch(i)

		if traceIsNull != isNull {
			internal.UpdateProgressHeight(i, false, "trace is null but tipset is not", db)
			continue
		}

		internal.UpdateProgressHeight(i, true, internal.ProgressOK, db)
	}
	return nil
}
