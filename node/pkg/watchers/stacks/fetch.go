package stacks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/certusone/wormhole/node/pkg/common"
)

type (
	// Partial types from https://github.com/stacks-network/stacks-core/blob/master/docs/rpc/openapi.yaml

	StacksStxTransferEvent struct {
		Amount    string `json:"amount"`
		Memo      string `json:"memo"`
		Recipient string `json:"recipient"`
		Sender    string `json:"sender"`
	}

	StacksContractEvent struct {
		ContractIdentifier string                 `json:"contract_identifier"`
		RawValue           string                 `json:"raw_value"`
		Topic              string                 `json:"topic"`
		Value              map[string]interface{} `json:"value"`
	}

	StacksEvent struct {
		Committed        bool                    `json:"committed"`
		EventIndex       uint64                  `json:"event_index"`
		TxID             string                  `json:"txid"`
		Type             string                  `json:"type"`
		StxTransferEvent *StacksStxTransferEvent `json:"stx_transfer_event,omitempty"`
		ContractEvent    *StacksContractEvent    `json:"contract_event,omitempty"`
	}

	StacksV3TenureBlock struct {
		BlockId       string `json:"block_id"`
		BlockHash     string `json:"block_hash"`
		ParentBlockId string `json:"parent_block_id"`
		Height        uint64 `json:"height"`
	}

	StacksV3TenureBlocksResponse struct {
		ConsensusHash   string                `json:"consensus_hash"`
		BurnBlockHeight uint64                `json:"burn_block_height"`
		BurnBlockHash   string                `json:"burn_block_hash"`
		StacksBlocks    []StacksV3TenureBlock `json:"stacks_blocks"`
	}

	StacksV3TenureBlockTransaction struct {
		TxId                 string                 `json:"txid"`
		TxIndex              uint32                 `json:"tx_index"`                         // Warning: May default to 0 for replay endpoint
		Data                 map[string]interface{} `json:"data,omitempty"`                   // Transaction data structure
		Hex                  string                 `json:"hex,omitempty"`                    // Raw transaction hex
		ResultHex            string                 `json:"result_hex,omitempty"`             // Transaction execution result in hex
		Events               []StacksEvent          `json:"events,omitempty"`                 // Transaction events
		PostConditionAborted bool                   `json:"post_condition_aborted,omitempty"` // Whether the post-condition was aborted
		VmError              *string                `json:"vm_error,omitempty"`               // Runtime error message if transaction failed (null when successful)
	}

	StacksV3TenureBlockReplayResponse struct {
		BlockId         string                           `json:"block_id"`
		BlockHash       string                           `json:"block_hash"`
		BlockHeight     uint64                           `json:"block_height"`
		ParentBlockId   string                           `json:"parent_block_id"`
		ConsensusHash   string                           `json:"consensus_hash"`
		TxMerkleRoot    string                           `json:"tx_merkle_root"`
		StateIndexRoot  string                           `json:"state_index_root"`
		Timestamp       uint64                           `json:"timestamp"`
		MinerSignature  string                           `json:"miner_signature"`
		SignerSignature []string                         `json:"signer_signature"`
		Transactions    []StacksV3TenureBlockTransaction `json:"transactions"`
		ValidMerkleRoot bool                             `json:"valid_merkle_root"`
	}

	StacksV3TransactionResponse struct {
		IndexBlockHash string  `json:"index_block_hash"`
		Tx             string  `json:"tx"`
		Result         string  `json:"result"`
		BlockHeight    *uint64 `json:"block_height"`
		IsCanonical    bool    `json:"is_canonical"`
	}

	StacksV2PoxEpoch struct {
		EpochID     string `json:"epoch_id"`
		StartHeight uint64 `json:"start_height"`
		EndHeight   uint64 `json:"end_height"`
	}

	StacksV2PoxResponse struct {
		ContractID                  string             `json:"contract_id"`
		FirstBurnchainBlockHeight   uint64             `json:"first_burnchain_block_height"`
		CurrentBurnchainBlockHeight uint64             `json:"current_burnchain_block_height"`
		PreparePhaseBlockLength     uint64             `json:"prepare_phase_block_length"`
		RewardPhaseBlockLength      uint64             `json:"reward_phase_block_length"`
		Epochs                      []StacksV2PoxEpoch `json:"epochs"`
	}

	StacksV2InfoPoxAnchor struct {
		AnchorBlockHash string `json:"anchor_block_hash"`
		AnchorBlockTxid string `json:"anchor_block_txid"`
	}

	StacksV2InfoResponse struct {
		PeerVersion            uint32                 `json:"peer_version"`
		PoxConsensus           string                 `json:"pox_consensus"`
		BurnBlockHeight        uint64                 `json:"burn_block_height"`
		StablePoxConsensus     string                 `json:"stable_pox_consensus"`
		StableBurnBlockHeight  uint64                 `json:"stable_burn_block_height"`
		ServerVersion          string                 `json:"server_version"`
		NetworkID              uint32                 `json:"network_id"`
		ParentNetworkID        uint32                 `json:"parent_network_id"`
		StacksTipHeight        uint64                 `json:"stacks_tip_height"`
		StacksTip              string                 `json:"stacks_tip"`
		StacksTipConsensusHash string                 `json:"stacks_tip_consensus_hash"`
		GenesisChainStateHash  string                 `json:"genesis_chainstate_hash"`
		UnanchoredTip          *string                `json:"unanchored_tip"`
		UnanchoredSeq          *uint16                `json:"unanchored_seq"`
		TenureHeight           uint64                 `json:"tenure_height"`
		ExitAtBlockHeight      *uint64                `json:"exit_at_block_height"`
		IsFullySynced          bool                   `json:"is_fully_synced"`
		NodePublicKey          string                 `json:"node_public_key"`
		NodePublicKeyHash      string                 `json:"node_public_key_hash"`
		LastPoxAnchor          *StacksV2InfoPoxAnchor `json:"last_pox_anchor"`
		Stackerdbs             []string               `json:"stackerdbs"`
	}
)

// Fetches a tenure and its blocks by Bitcoin (burn) block height
func (w *Watcher) fetchTenureBlocksByBurnHeight(ctx context.Context, height uint64) (*StacksV3TenureBlocksResponse, error) {
	url := fmt.Sprintf("%s/v3/tenures/blocks/height/%d", w.rpcURL, height)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Bitcoin (burn) block: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Burn block was skipped, return empty tenure blocks
		return &StacksV3TenureBlocksResponse{
			BurnBlockHeight: height,
			StacksBlocks:    []StacksV3TenureBlock{},
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tenureBlocks StacksV3TenureBlocksResponse
	if err := json.Unmarshal(body, &tenureBlocks); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &tenureBlocks, nil
}

// Fetches block replay data including all transactions for a given block
// Uses the v3 blocks/replay endpoint which includes vm_error for failed transactions
func (w *Watcher) fetchStacksBlockReplay(ctx context.Context, blockId string) (*StacksV3TenureBlockReplayResponse, error) {
	url := fmt.Sprintf("%s/v3/blocks/replay/%s", w.rpcURL, blockId)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// The replay endpoint requires an authorized request, as it's protected due to potential resource concerns.
	resp, err := w.doAuthorizedRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block replay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var replay StacksV3TenureBlockReplayResponse
	if err := json.Unmarshal(body, &replay); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &replay, nil
}

// Fetches a transaction by its txid
func (w *Watcher) fetchStacksTransactionByTxId(ctx context.Context, txID string) (*StacksV3TransactionResponse, error) {
	txID = strings.TrimPrefix(txID, "0x")
	url := fmt.Sprintf("%s/v3/transaction/%s", w.rpcURL, txID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transaction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var tx StacksV3TransactionResponse
	if err := json.Unmarshal(body, &tx); err != nil {
		return nil, fmt.Errorf("failed to parse node transaction response: %w", err)
	}

	return &tx, nil
}

// Fetches PoX (Proof of Transfer) information including epoch data
func (w *Watcher) fetchPoxInfo(ctx context.Context) (*StacksV2PoxResponse, error) {
	url := fmt.Sprintf("%s/v2/pox", w.rpcURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PoX info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var poxInfo StacksV2PoxResponse
	if err := json.Unmarshal(body, &poxInfo); err != nil {
		return nil, fmt.Errorf("failed to parse PoX info response: %w", err)
	}

	return &poxInfo, nil
}

// Fetches node information from the Stacks node
func (w *Watcher) fetchNodeInfo(ctx context.Context) (*StacksV2InfoResponse, error) {
	url := fmt.Sprintf("%s/v2/info", w.rpcURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := w.doRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Stacks node info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := common.SafeRead(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var nodeInfo StacksV2InfoResponse
	if err := json.Unmarshal(body, &nodeInfo); err != nil {
		return nil, fmt.Errorf("failed to parse Stacks node info response: %w", err)
	}

	return &nodeInfo, nil
}
