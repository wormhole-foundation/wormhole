package cli

import (
	"context"

	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func CmdShowActiveGuardianSetIndex() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-active-guardian-set-index",
		Short: "shows active-guardian-set-index",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryGetActiveGuardianSetIndexRequest{}

			res, err := queryClient.ActiveGuardianSetIndex(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
