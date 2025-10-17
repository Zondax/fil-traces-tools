package cmd

import (
	"fmt"

	"github.com/bytedance/sonic"
	apitypes "github.com/filecoin-project/lotus/api"
	"github.com/spf13/cobra"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal"
	"go.uber.org/zap"
)

func ValidateJSONCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   internal.ValidateJSONCheck,
		Short: "Validate JSON",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return validateJSON(cmd)
		},
	}
	cmd.Flags().Int64(internal.StartFlag, 1, "start height to validate")
	cmd.Flags().Int64(internal.EndFlag, 100, "end height to validate")
	cmd.Flags().String(internal.DBPathFlag, ".", "path to the database")
	return cmd
}

func validateJSON(cmd *cobra.Command) error {
	config := api.GetGlobalConfigs()
	log := initLogger()
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
	db, err := api.NewDB(dbPath, internal.ValidateJSONCheck)
	if err != nil {
		log.Error("failed to create db", zap.Error(err))
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close database", zap.Error(err))
		}
	}()

	dataStore, err := api.GetDataStoreClient(&config)
	if err != nil {
		log.Error("failed to get data store client", zap.Error(err))
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
		log.Debug(fmt.Sprintf("Validating JSON for height %d", i))
		data, err := api.GetTraceFromDataStore(i, dataStore, &config)
		if err != nil {
			log.Error("failed to get trace from data store", zap.Error(err))
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		var computeState apitypes.ComputeStateOutput
		err = sonic.Unmarshal(data, &computeState)
		if err != nil {
			internal.UpdateProgressHeight(i, false, err.Error(), db)
			continue
		}
		internal.UpdateProgressHeight(i, true, internal.ProgressOK, db)
	}
	return nil
}
