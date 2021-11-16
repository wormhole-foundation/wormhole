package cli

import (
	"context"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
)

func CmdListChainRegistration() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-chain-registration",
		Short: "list all ChainRegistration",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllChainRegistrationRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.ChainRegistrationAll(context.Background(), params)
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

func CmdShowChainRegistration() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-chain-registration [chainID]",
		Short: "shows a ChainRegistration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			chainID, err := strconv.ParseUint(args[0], 10, 16)
			if err != nil {
				return err
			}

			params := &types.QueryGetChainRegistrationRequest{
				ChainID: uint32(chainID),
			}

			res, err := queryClient.ChainRegistration(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
