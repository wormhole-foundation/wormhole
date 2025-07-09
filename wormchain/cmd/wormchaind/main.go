package main

import (
	"os"

	"cosmossdk.io/log"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/wormhole-foundation/wormchain/app"
)

func main() {
	app.SetAddressPrefixes()
	rootCmd, _ := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "WORMCHAIND", app.DefaultNodeHome); err != nil {
		log.NewLogger(rootCmd.OutOrStderr()).Error("failure when running app", "err", err)
		os.Exit(1)
	}
}
