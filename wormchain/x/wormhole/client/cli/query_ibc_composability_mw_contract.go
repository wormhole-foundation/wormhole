package cli

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func CmdShowIbcComposabilityMwContract() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-ibc-composability-mw-contract",
		Short: "show the contract that is used by the ibc composability middleware",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			req := &types.QueryIbcComposabilityMwContractRequest{}

			res, err := queryClient.IbcComposabilityMwContract(context.Background(), req)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
