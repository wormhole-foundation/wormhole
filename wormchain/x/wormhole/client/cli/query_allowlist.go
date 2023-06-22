package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func CmdListAllowlists() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-allowlists",
		Short: "list all allowlists created by validators",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllValidatorAllowlist{
				Pagination: pageReq,
			}

			res, err := queryClient.AllowlistAll(context.Background(), params)
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

func CmdShowAllowlist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-allowlist [validator-address]",
		Short: "shows an allowlist owned by a specific validator address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)
			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			argValidatorAddress := args[0]

			params := &types.QueryValidatorAllowlist{
				ValidatorAddress: argValidatorAddress,
				Pagination:       pageReq,
			}

			res, err := queryClient.Allowlist(context.Background(), params)
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
