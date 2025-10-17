package cmd

import (
	"fmt"

	"github.com/bytedance/sonic"
	address "github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	apitypes "github.com/filecoin-project/lotus/api"
	"github.com/spf13/cobra"
	"github.com/zondax/fil-parser/actors/v2/reward"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal"
	"go.uber.org/zap"
)

const (
	methodAwardBlockReward = abi.MethodNum(2)
	rewardActorAddr        = "f02"
	network                = "mainnet"
	paramKey               = "Params"
)

func ValidateCanonicalChainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   internal.CanonicalChainCheck,
		Short: "Validate canonical chain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return validateCanonicalChain(cmd)
		},
	}
	cmd.Flags().Int64(internal.StartFlag, 1, "start height to validate")
	cmd.Flags().Int64(internal.EndFlag, 100, "end height to validate")
	cmd.Flags().String(internal.DBPathFlag, ".", "path to the database")
	return cmd
}

func validateCanonicalChain(cmd *cobra.Command) error {
	config := api.GetGlobalConfigs()
	log := initLogger()
	ctx := cmd.Context()

	start, err := cmd.Flags().GetInt64(internal.StartFlag)
	if err != nil {
		log.Error("could not get start flag", zap.Error(err))
		return err
	}
	end, err := cmd.Flags().GetInt64(internal.EndFlag)
	if err != nil {
		log.Error("could not get end flag", zap.Error(err))
		return err
	}
	dbPath, err := cmd.Flags().GetString(internal.DBPathFlag)
	if err != nil {
		log.Error("could not get db-path flag", zap.Error(err))
		return err
	}
	db, err := api.NewDB(dbPath, internal.CanonicalChainCheck)
	if err != nil {
		log.Error("could not create db", zap.Error(err))
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close database", zap.Error(err))
		}
	}()

	rpcClient, err := api.NewFilecoinRPCClient(ctx, config.NodeURL, config.NodeToken)
	if err != nil {
		log.Error("could not create rpc client", zap.Error(err))
		return err
	}
	dataStore, err := api.GetDataStoreClient(&config)
	if err != nil {
		log.Error("could not create data store client", zap.Error(err))
		return err
	}
	rewardActor := &reward.Reward{}

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
		log.Debug(fmt.Sprintf("Validating canonical chain for height %d", i))

		data, err := api.GetTraceFromDataStore(i, dataStore, &config)
		if err != nil {
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		var computeState apitypes.ComputeStateOutput
		err = sonic.Unmarshal(data, &computeState)
		if err != nil {
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		// get miners
		traceMiners := map[string]bool{}
		for _, trace := range computeState.Trace {
			if trace.Msg.To.String() == rewardActorAddr && trace.Msg.Method == methodAwardBlockReward {
				parsedParams, err := rewardActor.AwardBlockReward(network, i, trace.Msg.Params)
				if err != nil {
					log.Error(fmt.Sprintf("could not parse parameters for height: %d", i), zap.Error(err))
					continue
				}
				// Get the miner that received the reward
				params, ok := parsedParams[paramKey]
				if !ok {
					log.Error(fmt.Sprintf("could not get parameter '%s' for height: %d", paramKey, i), zap.Error(err))
					continue
				}
				miner := reward.GetMinerFromAwardBlockRewardParams(params)
				if miner == "" {
					log.Error(fmt.Sprintf("found empty miner for height: %d", i), zap.Error(err))
					continue
				}
				traceMiners[miner] = true
			}
		}

		onchainMiners := map[string]bool{}
		tipset, err := api.ChainGetTipSetByHeight(ctx, i, rpcClient)
		if err != nil {
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		blocks := tipset.Blocks()
		for _, block := range blocks {
			onchainMiners[block.Miner.String()] = true
		}
		// check that the length of miners are the same
		if len(traceMiners) != len(onchainMiners) {
			internal.UpdateProgressHeight(i, false, "length of miners do not match", db)
			continue
		}

		// check that the miners are the same ( including equivalent addresses )
		for miner := range traceMiners {
			// get equivalent addresses for the miner
			minerAddr, err := address.NewFromString(miner)
			if err != nil {
				log.Error(fmt.Sprintf("could not create address for miner %s at height %d", miner, i), zap.Error(err))
				continue
			}
			equivalentAddresses, err := internal.GetEquivalentAddresses(ctx, minerAddr, rpcClient.FullNodeClient())
			if err != nil {
				log.Error(fmt.Sprintf("could not get equivalent addresses for miner %s at height %d", miner, i), zap.Error(err))
				continue
			}
			var found bool
			for equivalentAddress := range equivalentAddresses {
				if _, ok := onchainMiners[equivalentAddress]; ok {
					found = true
					break
				}
			}
			if !found {
				internal.UpdateProgressHeight(i, false, fmt.Sprintf("miner %s not found", miner), db)
				continue
			}
		}
		internal.UpdateProgressHeight(i, true, "ok", db)
	}
	return nil
}
