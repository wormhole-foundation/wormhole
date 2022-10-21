package cli

import (
	"context"

	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

func CmdListGuardianValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list-guardian-validator",
		Short: "list all guardian-validator",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			pageReq, err := client.ReadPageRequest(cmd.Flags())
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryAllGuardianValidatorRequest{
				Pagination: pageReq,
			}

			res, err := queryClient.GuardianValidatorAll(context.Background(), params)
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

func CmdShowGuardianValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show-guardian-validator [guardian-key]",
		Short: "shows a guardian-validator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx := client.GetClientContextFromCmd(cmd)

			queryClient := types.NewQueryClient(clientCtx)

			argGuardianKey, err := hex.DecodeString(args[0])

			if err != nil {
				return err
			}

			params := &types.QueryGetGuardianValidatorRequest{
				GuardianKey: argGuardianKey,
			}

			res, err := queryClient.GuardianValidator(context.Background(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
