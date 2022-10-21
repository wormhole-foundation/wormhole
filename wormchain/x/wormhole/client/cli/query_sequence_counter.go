package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func CmdListSequenceCounter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-sequence-counter",
		Short: "list all SequenceCounter",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllSequenceCounterRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.SequenceCounterAll(context.Background(), params)
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

func CmdShowSequenceCounter() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-sequence-counter [index]",
		Short: "shows a SequenceCounter",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argIndex := args[0]

			params := &types.QueryGetSequenceCounterRequest{
				Index: argIndex,
			}

			res, err := queryClient.SequenceCounter(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
