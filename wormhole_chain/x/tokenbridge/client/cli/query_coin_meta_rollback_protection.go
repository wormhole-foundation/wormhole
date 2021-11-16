package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

func CmdListCoinMetaRollbackProtection() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-coin-meta-rollback-protection",
		Short: "list all CoinMetaRollbackProtection",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllCoinMetaRollbackProtectionRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.CoinMetaRollbackProtectionAll(context.Background(), params)
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

func CmdShowCoinMetaRollbackProtection() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-coin-meta-rollback-protection [index]",
		Short: "shows a CoinMetaRollbackProtection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetCoinMetaRollbackProtectionRequest{
				Index: argIndex,
			}

			res, err := queryClient.CoinMetaRollbackProtection(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
