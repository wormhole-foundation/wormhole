package cli

import (
	"strconv"

	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

var _ = strconv.Itoa(0)

func CmdLatestGuardianSetIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "latest-guardian-set-index",
		Short: "Query latest_guardian_set_index",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryLatestGuardianSetIndexRequest{}

			res, err := queryClient.LatestGuardianSetIndex(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
