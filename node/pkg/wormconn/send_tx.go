package wormconn

import (
	"context"
	"fmt"
	"time"

	txclient "github.com/cosmos/cosmos-sdk/client/tx"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	sdktx "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func (c *ClientConn) SignAndBroadcastTx(ctx context.Context, msg sdktypes.Msg) (*sdktx.BroadcastTxResponse, error) {
	// Lock to protect the wallet sequence number.
	c.mutex.Lock()
	defer c.mutex.Unlock()

	authClient := auth.NewQueryClient(c.c)
	accountQuery := &auth.QueryAccountRequest{
		Address: c.senderAddress,
	}
	resp, err := authClient.Account(ctx, accountQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch account: %w", err)
	}

	var account auth.AccountI
	if err := c.encCfg.InterfaceRegistry.UnpackAny(resp.Account, &account); err != nil {
		return nil, fmt.Errorf("failed to unmarshal account info: %w", err)
	}

	builder := c.encCfg.TxConfig.NewTxBuilder()
	if err := builder.SetMsgs(msg); err != nil {
		return nil, fmt.Errorf("failed to add message to builder: %w", err)
	}
	builder.SetGasLimit(2000000) // TODO: Maybe simulate and use the result

	// The tx needs to be signed in 2 passes: first we populate the SignerInfo
	// inside the TxBuilder and then sign the payload.
	sequence := account.GetSequence()
	sig := signing.SignatureV2{
		PubKey: c.privateKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  c.encCfg.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: sequence,
	}
	if err := builder.SetSignatures(sig); err != nil {
		return nil, fmt.Errorf("failed to set SignerInfo: %w", err)
	}

	signerData := authsigning.SignerData{
		ChainID:       c.chainId,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      sequence,
	}

	sig, err = txclient.SignWithPrivKey(
		c.encCfg.TxConfig.SignModeHandler().DefaultMode(),
		signerData,
		builder,
		c.privateKey,
		c.encCfg.TxConfig,
		sequence,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to sign tx: %w", err)
	}
	if err := builder.SetSignatures(sig); err != nil {
		return nil, fmt.Errorf("failed to update tx signature: %w", err)
	}

	txBytes, err := c.encCfg.TxConfig.TxEncoder()(builder.GetTx())
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tx: %w", err)
	}

	client := sdktx.NewServiceClient(c.c)

	// Returns *BroadcastTxResponse
	txResp, err := client.BroadcastTx(
		ctx,
		&sdktx.BroadcastTxRequest{
			Mode:    sdktx.BroadcastMode_BROADCAST_MODE_SYNC,
			TxBytes: txBytes,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast tx: %w", err)
	}

	// Wait for the tx to be included in a block (13 seconds for 2 blocks minimum)
	res, err := waitForBlockInclusion(ctx, client, txResp.TxResponse.TxHash, 13*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for tx inclusion: %w", err)
	} else {

		fmt.Println("JOEL - TX BROADCASTED BEFORE", txResp.TxResponse)

		// update the response with the final result
		txResp.TxResponse = res.TxResponse

		fmt.Println("JOEL - TX BROADCASTED AFTER", txResp.TxResponse)
	}

	return txResp, nil
}

// waitForBlockInclusion waits for the tx to be included in a block, or times out after a given duration.
func waitForBlockInclusion(ctx context.Context, client sdktx.ServiceClient, txHash string, waitTimeout time.Duration) (*sdktx.GetTxResponse, error) {
	exitAfter := time.After(waitTimeout)
	for {
		select {
		// check if wait timeout is exceeded
		case <-exitAfter:
			fmt.Println("JOEL - TIMED OUT", txHash)
			return nil, fmt.Errorf("timed out after: %d; wait for tx %s to be included in a block", waitTimeout, txHash)
		// check if in block every second
		case <-time.After(1 * time.Second):
			res, err := client.GetTx(ctx, &sdktx.GetTxRequest{Hash: txHash})
			if err == nil {
				fmt.Print("JOEL - FOUND TX IN BLOCK", txHash)
				return res, nil
			}
		// check if context is done
		case <-ctx.Done():
			fmt.Println("JOEL - CONTEXT DONE", txHash, ctx.Err())
			return nil, ctx.Err()
		}
	}
}
