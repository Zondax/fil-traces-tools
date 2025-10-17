package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/internal"
	"go.uber.org/zap"
)

func GenerateReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-report",
		Short: "Generate report",
		Long: `Generate report for a specific check
				Supported checks:
					- validate-null-blocks
					- validate-json
					- validate-canonical-chain
					- validate-address-balance
					- validate-multisig-state
					- validate-address-balance-sequential
					- validate-multisig-state-sequential
		`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return generateReport(cmd)
		},
	}

	cmd.Flags().String(internal.DBPathFlag, ".", "--db-path .")
	cmd.Flags().String(internal.ReportPathFlag, ".", "--report-path .")
	cmd.Flags().String(internal.CheckFlag, "", "--check <check>")
	return cmd
}

var availableChecks = map[string]bool{
	internal.NullBlocksCheck:               true,
	internal.ValidateJSONCheck:             true,
	internal.CanonicalChainCheck:           true,
	internal.AddressBalanceCheck:           true,
	internal.MultisigStateCheck:            true,
	internal.AddressBalanceSequentialCheck: true,
	internal.MultisigStateSequentialCheck:  true,
}

func generateReport(cmd *cobra.Command) error {
	log := initLogger()

	check, err := cmd.Flags().GetString(internal.CheckFlag)
	if err != nil {
		log.Error("failed to get check", zap.Error(err))
		return err
	}
	if _, ok := availableChecks[check]; !ok {
		log.Error("invalid check, expected one of: validate-null-blocks, validate-json, validate-canonical-chain, validate-address-balance, validate-multisig-state, validate-address-balance-sequential, validate-multisig-state-sequential", zap.String("check", check))
		return err
	}
	reportPath, err := cmd.Flags().GetString(internal.ReportPathFlag)
	if err != nil {
		log.Error("failed to get report path", zap.Error(err))
		return err
	}
	dbPath, err := cmd.Flags().GetString(internal.DBPathFlag)
	if err != nil {
		log.Error("failed to get db path", zap.Error(err))
		return err
	}
	db, err := api.NewDB(dbPath, check)
	if err != nil {
		log.Error("failed to open db", zap.Error(err))
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close database", zap.Error(err))
		}
	}()

	log.Info("generating report", zap.String("check", check), zap.String("report-path", reportPath))
	data, err := db.GetAllKVAsJSON()
	if err != nil {
		log.Error("failed to get all kv as json", zap.Error(err))
		return err
	}

	if err := os.WriteFile(reportPath, data, 0600); err != nil {
		log.Error("failed to write report", zap.Error(err))
		return err
	}
	log.Info("report generated", zap.String("report-path", reportPath))

	return nil
}
