package rest

import (
	"encoding/hex"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

type (
	// GuardianSetUpdateProposalReq defines a guardian set update proposal request body.
	GuardianSetUpdateProposalReq struct {
		BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

		Title            string         `json:"title" yaml:"title"`
		Description      string         `json:"description" yaml:"description"`
		GuardianSetIndex uint32         `json:"guardianSetIndex" yaml:"guardianSetIndex"`
		GuardianSetKeys  []string       `json:"guardianSetKeys" yaml:"guardianSetKeys"`
		Proposer         sdk.AccAddress `json:"proposer" yaml:"proposer"`
		Deposit          sdk.Coins      `json:"deposit" yaml:"deposit"`
	}

	// WormholeGovernanceMessageProposalReq defines a wormhole governance message proposal request body.
	WormholeGovernanceMessageProposalReq struct {
		BaseReq rest.BaseReq `json:"base_req" yaml:"base_req"`

		Title       string         `json:"title" yaml:"title"`
		Description string         `json:"description" yaml:"description"`
		TargetChain uint16         `json:"targetChain" yaml:"targetChain"`
		Action      uint8          `json:"action" yaml:"action"`
		Module      []byte         `json:"module" yaml:"module"`
		Payload     []byte         `json:"payload" yaml:"payload"`
		Proposer    sdk.AccAddress `json:"proposer" yaml:"proposer"`
		Deposit     sdk.Coins      `json:"deposit" yaml:"deposit"`
	}
)

// ProposalGuardianSetUpdateRESTHandler returns a ProposalRESTHandler that exposes the guardian set update
// REST handler with a given sub-route.
func ProposalGuardianSetUpdateRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "wormhole_guardian_update",
		Handler:  postProposalGuardianSetUpdateHandlerFn(clientCtx),
	}
}

func postProposalGuardianSetUpdateHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GuardianSetUpdateProposalReq
		if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		keys := make([][]byte, len(req.GuardianSetKeys))
		for i, keyString := range req.GuardianSetKeys {
			keyBytes, err := hex.DecodeString(keyString)
			if rest.CheckBadRequestError(w, err) {
				return
			}
			keys[i] = keyBytes
		}

		content := types.NewGuardianSetUpdateProposal(req.Title, req.Description, types.GuardianSet{
			Index:          req.GuardianSetIndex,
			Keys:           keys,
			ExpirationTime: 0,
		})

		msg, err := govtypes.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
		if rest.CheckBadRequestError(w, err) {
			return
		}
		if rest.CheckBadRequestError(w, msg.ValidateBasic()) {
			return
		}

		tx.WriteGeneratedTxResponse(clientCtx, w, req.BaseReq, msg)
	}
}

// ProposalWormholeGovernanceMessageRESTHandler returns a ProposalRESTHandler that exposes the wormhole governance message
// REST handler with a given sub-route.
func ProposalWormholeGovernanceMessageRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "wormhole_governance_message",
		Handler:  postProposalWormholeGovernanceMessageHandlerFn(clientCtx),
	}
}

func postProposalWormholeGovernanceMessageHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req WormholeGovernanceMessageProposalReq
		if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		content := types.NewGovernanceWormholeMessageProposal(req.Title, req.Description, req.Action, req.TargetChain, req.Module, req.Payload)

		msg, err := govtypes.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
		if rest.CheckBadRequestError(w, err) {
			return
		}
		if rest.CheckBadRequestError(w, msg.ValidateBasic()) {
			return
		}

		tx.WriteGeneratedTxResponse(clientCtx, w, req.BaseReq, msg)
	}
}
