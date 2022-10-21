package cli

import (
	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/client/cli"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

const FlagGuardianSetKeys = "guardian-set-keys"
const FlagGuardianSetIndex = "guardian-set-index"

// NewCmdSubmitGuardianSetUpdateProposal implements a command handler for submitting a guardian set update governance
// proposal.
func NewCmdSubmitGuardianSetUpdateProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update-guardian-set [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a guardian set update proposal",
		Long:  "Submit a proposal to update the current guardian set to a new one",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return err
			}

			keyStrings, err := cmd.Flags().GetStringArray(FlagGuardianSetKeys)
			if err != nil {
				return err
			}

			newIndex, err := cmd.Flags().GetUint32(FlagGuardianSetIndex)
			if err != nil {
				return err
			}

			keys := make([][]byte, len(keyStrings))
			for i, keyString := range keyStrings {
				keyBytes, err := hex.DecodeString(keyString)
				if err != nil {
					return err
				}
				keys[i] = keyBytes
			}

			content := types.NewGuardianSetUpdateProposal(title, description, types.GuardianSet{
				Index:          newIndex,
				Keys:           keys,
				ExpirationTime: 0,
			})
			err = content.ValidateBasic()
			if err != nil {
				return err
			}

			msg, err := gov.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().StringArray(FlagGuardianSetKeys, []string{}, "list of guardian keys (hex encoded without 0x)")
	cmd.Flags().Uint32(FlagGuardianSetIndex, 0, "index of the new guardian set")
	cmd.MarkFlagRequired(cli.FlagTitle)
	cmd.MarkFlagRequired(cli.FlagDescription)
	cmd.MarkFlagRequired(FlagGuardianSetKeys)
	cmd.MarkFlagRequired(FlagGuardianSetIndex)

	return cmd
}

const FlagAction = "action"
const FlagModule = "module"
const FlagTargetChainID = "target-chain-id"
const FlagPayload = "payload"

// NewCmdSubmitWormholeGovernanceMessageProposal implements a command handler for submitting a generic Wormhole
// governance message.
func NewCmdSubmitWormholeGovernanceMessageProposal() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wormhole-governance-message [flags]",
		Args:  cobra.ExactArgs(0),
		Short: "Submit a wormhole governance message proposal",
		Long:  "Submit a proposal to emit a generic wormhole governance message",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			from := clientCtx.GetFromAddress()

			depositStr, err := cmd.Flags().GetString(cli.FlagDeposit)
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(depositStr)
			if err != nil {
				return err
			}

			title, err := cmd.Flags().GetString(cli.FlagTitle)
			if err != nil {
				return err
			}

			description, err := cmd.Flags().GetString(cli.FlagDescription)
			if err != nil {
				return err
			}

			action, err := cmd.Flags().GetUint8(FlagAction)
			if err != nil {
				return err
			}

			targetChain, err := cmd.Flags().GetUint16(FlagTargetChainID)
			if err != nil {
				return err
			}

			module, err := cmd.Flags().GetBytesHex(FlagModule)
			if err != nil {
				return err
			}

			payload, err := cmd.Flags().GetBytesHex(FlagPayload)
			if err != nil {
				return err
			}

			content := types.NewGovernanceWormholeMessageProposal(title, description, action, targetChain, module, payload)
			err = content.ValidateBasic()
			if err != nil {
				return err
			}

			msg, err := gov.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().String(cli.FlagTitle, "", "title of proposal")
	cmd.Flags().String(cli.FlagDescription, "", "description of proposal")
	cmd.Flags().String(cli.FlagDeposit, "", "deposit of proposal")
	cmd.Flags().Uint8(FlagAction, 0, "target chain of the message (0 for all)")
	cmd.Flags().Uint16(FlagTargetChainID, 0, "target chain of the message (0 for all)")
	cmd.Flags().BytesHex(FlagModule, []byte{}, "module identifier of the message")
	cmd.Flags().BytesHex(FlagPayload, []byte{}, "payload of the message")
	cmd.MarkFlagRequired(cli.FlagTitle)
	cmd.MarkFlagRequired(cli.FlagDescription)
	cmd.MarkFlagRequired(FlagAction)
	cmd.MarkFlagRequired(FlagTargetChainID)
	cmd.MarkFlagRequired(FlagModule)
	cmd.MarkFlagRequired(FlagPayload)

	return cmd
}
