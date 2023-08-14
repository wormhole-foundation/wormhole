package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/ibc-hooks/keeper"

	"github.com/wormhole-foundation/wormchain/x/ibc-hooks/types"
)

func indexRunCmd(cmd *cobra.Command, args []string) error {
	usageTemplate := `Usage:{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}
  
{{if .HasAvailableSubCommands}}Available Commands:{{range .Commands}}{{if .IsAvailableCommand}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
	cmd.SetUsageTemplate(usageTemplate)
	return cmd.Help()
}

// GetQueryCmd returns the cli query commands for this module.
func GetQueryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       indexRunCmd,
	}

	cmd.AddCommand(
		GetCmdWasmSender(),
	)
	return cmd
}

// GetCmdPoolParams return pool params.
func GetCmdWasmSender() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wasm-sender <channelID> <originalSender>",
		Short: "Generate the local address for a wasm hooks sender",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Generate the local address for a wasm hooks sender.
Example:
$ %s query ibc-hooks wasm-hooks-sender channel-42 juno12smx2wdlyttvyzvzg54y2vnqwq2qjatezqwqxu
`,
				version.AppName,
			),
		),
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			channelID := args[0]
			originalSender := args[1]
			// ToDo: Make this flexible as an arg
			prefix := sdk.GetConfig().GetBech32AccountAddrPrefix()
			senderBech32, err := keeper.DeriveIntermediateSender(channelID, originalSender, prefix)
			if err != nil {
				return err
			}
			fmt.Println(senderBech32)
			return nil
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
