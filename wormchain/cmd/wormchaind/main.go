package main

import (
	"os"

	"cosmossdk.io/log"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/wormhole-foundation/wormchain/app"
	cmd "github.com/wormhole-foundation/wormchain/cmd/wormchaind/cmd"
)

func main() {
	// TODO: JOEL - LOOK INTO SETTING ADDRESS PREFIXES
	app.SetAddressPrefixes()
	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "WORMCHAIND", app.DefaultNodeHome); err != nil {
		log.NewLogger(rootCmd.OutOrStderr()).Error("failure when running app", "err", err)
		os.Exit(1)
	}

	// TODO: JOEL - REMOVE BELOW
	// rootCmd, _ := cosmoscmd.NewRootCmd(
	// 	app.Name,
	// 	app.AccountAddressPrefix,
	// 	app.DefaultNodeHome,
	// 	app.Name,
	// 	app.ModuleBasics,
	// 	app.New,
	// 	// this line is used by starport scaffolding # root/arguments
	// )

	// rootCmd.AddCommand(cli.GetGenesisCmd())
	// if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
	// 	os.Exit(1)
	// }
}
