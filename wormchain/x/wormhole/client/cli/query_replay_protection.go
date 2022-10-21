package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func CmdListReplayProtection() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-replay-protection",
		Short: "list all ReplayProtection",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllReplayProtectionRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.ReplayProtectionAll(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddPaginationFlagsToCmd(cmd, cmd.Use)
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func CmdShowReplayProtection() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-replay-protection [index]",
		Short: "shows a ReplayProtection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetReplayProtectionRequest{
				Index: argIndex,
			}

			res, err := queryClient.ReplayProtection(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
