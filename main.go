package main

import (
	"github.com/zondax/fil-trace-check/api"
	"github.com/zondax/fil-trace-check/cmd"
	"github.com/zondax/golem/pkg/cli"
)

func main() {
	appSettings := cli.AppSettings{}

	cli := cli.New[*api.Config](appSettings)
	defer cli.Close()

	cli.GetRoot().AddCommand(cmd.ValidateNullBlocksCmd())
	cli.GetRoot().AddCommand(cmd.ValidateJSONCmd())
	cli.GetRoot().AddCommand(cmd.ValidateCanonicalChainCmd())
	cli.GetRoot().AddCommand(cmd.ValidateAddressBalanceCmd())
	cli.GetRoot().AddCommand(cmd.ValidateMultisigStateCmd())
	cli.GetRoot().AddCommand(cmd.GenerateReportCmd())
	cli.GetRoot().AddCommand(cmd.ValidateAddressBalanceSequentialCmd())
	cli.GetRoot().AddCommand(cmd.ValidateMultisigStateSequentialCmd())
	cli.Run()
}
