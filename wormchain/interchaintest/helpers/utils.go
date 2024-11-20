package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testreporter"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

type SearchEvent struct {
	Attribute string
	Value     string
}

func FindTxsByEvents(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, events []SearchEvent) TxsByEventsResp {
	var joinedEvents string
	for _, event := range events {
		joinedEvents += event.Attribute + "=" + event.Value + " "
	}

	stdout, stderr, err := chain.GetFullNode().ExecQuery(ctx, "txs", "--events", joinedEvents)
	require.NoError(t, err)

	t.Logf("STDERR: %s", string(stderr))
	t.Logf("Found txs: %s", string(stdout))

	res := new(TxsByEventsResp)
	err = json.Unmarshal(stdout, res)
	require.NoError(t, err)

	return *res
}

func MustAccAddressFromBech32(address string, bech32Prefix string) sdk.AccAddress {
	if len(strings.TrimSpace(address)) == 0 {
		panic("empty address string is not allowed")
	}

	bz, err := sdk.GetFromBech32(address, bech32Prefix)
	if err != nil {
		panic(err)
	}

	err = sdk.VerifyAddressFormat(bz)
	if err != nil {
		panic(err)
	}

	return sdk.AccAddress(bz)
}

func FindEventAttribute(t *testing.T, chain *cosmos.CosmosChain, txHash string, eventType string, attributeKey string, attributeValue string) bool {
	tx, err := chain.GetTransaction(txHash)
	require.NoError(t, err)
	for _, event := range tx.Events {
		if event.Type == eventType {
			for _, attribute := range event.Attributes {
				if string(attribute.Key) == attributeKey && string(attribute.Value) == attributeValue {
					fmt.Println("Found: ", eventType, " ", attributeKey, " ", attributeValue)
					return true
				}
			}
		}
	}
	fmt.Println("Not found: ", eventType, " ", attributeKey, " ", attributeValue, "!")
	return false
}

// FindOpenChannelByVersion queries all the channels of a given chain and returns the first with the given version. If no channel is found, it will fail the test.
func FindOpenChannelByVersion(
	t *testing.T,
	ctx context.Context,
	eRep *testreporter.RelayerExecReporter,
	r ibc.Relayer,
	chain *cosmos.CosmosChain,
	version string) ibc.ChannelOutput {

	// iterate up to 20 times to allow for chain to catch up
	for i := 0; i < 20; i++ {

		channels, err := r.GetChannels(ctx, eRep, chain.Config().ChainID)
		require.NoError(t, err)

		channelIdx := slices.IndexFunc(channels, func(channel ibc.ChannelOutput) bool {
			return channel.State == "STATE_OPEN" && channel.Version == version
		})
		if channelIdx != -1 {
			return channels[channelIdx]
		}
		testutil.WaitForBlocks(ctx, 1, chain)
	}

	require.Failf(t, "channel with version %s not found", version)
	return ibc.ChannelOutput{}
}

type TxsByEventsResp struct {
	TotalCount string `json:"total_count,omitempty"`
	Count      string `json:"count,omitempty"`
	PageNumber string `json:"page_number,omitempty"`
	PageTotal  string `json:"page_total,omitempty"`
	Limit      string `json:"limit,omitempty"`
	Txs        []struct {
		Height    string `json:"height,omitempty"`
		Txhash    string `json:"txhash,omitempty"`
		Codespace string `json:"codespace,omitempty"`
		Code      int    `json:"code,omitempty"`
		Data      string `json:"data,omitempty"`
		RawLog    string `json:"raw_log,omitempty"`
		Logs      []struct {
			MsgIndex int    `json:"msg_index,omitempty"`
			Log      string `json:"log,omitempty"`
			Events   []struct {
				Type       string `json:"type,omitempty"`
				Attributes []struct {
					Key   string `json:"key,omitempty"`
					Value string `json:"value,omitempty"`
				} `json:"attributes,omitempty"`
			} `json:"events,omitempty"`
		} `json:"logs,omitempty"`
		Info      string `json:"info,omitempty"`
		GasWanted string `json:"gas_wanted,omitempty"`
		GasUsed   string `json:"gas_used,omitempty"`
		Tx        struct {
			Type string `json:"@type,omitempty"`
			Body struct {
				Messages []struct {
					Type     string `json:"@type,omitempty"`
					ClientID string `json:"client_id,omitempty"`
					Header   struct {
						Type         string `json:"@type,omitempty"`
						SignedHeader struct {
							Header struct {
								Version struct {
									Block string `json:"block,omitempty"`
									App   string `json:"app,omitempty"`
								} `json:"version,omitempty"`
								ChainID     string    `json:"chain_id,omitempty"`
								Height      string    `json:"height,omitempty"`
								Time        time.Time `json:"time,omitempty"`
								LastBlockID struct {
									Hash          string `json:"hash,omitempty"`
									PartSetHeader struct {
										Total int    `json:"total,omitempty"`
										Hash  string `json:"hash,omitempty"`
									} `json:"part_set_header,omitempty"`
								} `json:"last_block_id,omitempty"`
								LastCommitHash     string `json:"last_commit_hash,omitempty"`
								DataHash           string `json:"data_hash,omitempty"`
								ValidatorsHash     string `json:"validators_hash,omitempty"`
								NextValidatorsHash string `json:"next_validators_hash,omitempty"`
								ConsensusHash      string `json:"consensus_hash,omitempty"`
								AppHash            string `json:"app_hash,omitempty"`
								LastResultsHash    string `json:"last_results_hash,omitempty"`
								EvidenceHash       string `json:"evidence_hash,omitempty"`
								ProposerAddress    string `json:"proposer_address,omitempty"`
							} `json:"header,omitempty"`
							Commit struct {
								Height  string `json:"height,omitempty"`
								Round   int    `json:"round,omitempty"`
								BlockID struct {
									Hash          string `json:"hash,omitempty"`
									PartSetHeader struct {
										Total int    `json:"total,omitempty"`
										Hash  string `json:"hash,omitempty"`
									} `json:"part_set_header,omitempty"`
								} `json:"block_id,omitempty"`
								Signatures []struct {
									BlockIDFlag      string    `json:"block_id_flag,omitempty"`
									ValidatorAddress string    `json:"validator_address,omitempty"`
									Timestamp        time.Time `json:"timestamp,omitempty"`
									Signature        string    `json:"signature,omitempty"`
								} `json:"signatures,omitempty"`
							} `json:"commit,omitempty"`
						} `json:"signed_header,omitempty"`
						ValidatorSet struct {
							Validators []struct {
								Address string `json:"address,omitempty"`
								PubKey  struct {
									Ed25519 string `json:"ed25519,omitempty"`
								} `json:"pub_key,omitempty"`
								VotingPower      string `json:"voting_power,omitempty"`
								ProposerPriority string `json:"proposer_priority,omitempty"`
							} `json:"validators,omitempty"`
							Proposer struct {
								Address string `json:"address,omitempty"`
								PubKey  struct {
									Ed25519 string `json:"ed25519,omitempty"`
								} `json:"pub_key,omitempty"`
								VotingPower      string `json:"voting_power,omitempty"`
								ProposerPriority string `json:"proposer_priority,omitempty"`
							} `json:"proposer,omitempty"`
							TotalVotingPower string `json:"total_voting_power,omitempty"`
						} `json:"validator_set,omitempty"`
						TrustedHeight struct {
							RevisionNumber string `json:"revision_number,omitempty"`
							RevisionHeight string `json:"revision_height,omitempty"`
						} `json:"trusted_height,omitempty"`
						TrustedValidators struct {
							Validators []struct {
								Address string `json:"address,omitempty"`
								PubKey  struct {
									Ed25519 string `json:"ed25519,omitempty"`
								} `json:"pub_key,omitempty"`
								VotingPower      string `json:"voting_power,omitempty"`
								ProposerPriority string `json:"proposer_priority,omitempty"`
							} `json:"validators,omitempty"`
							Proposer struct {
								Address string `json:"address,omitempty"`
								PubKey  struct {
									Ed25519 string `json:"ed25519,omitempty"`
								} `json:"pub_key,omitempty"`
								VotingPower      string `json:"voting_power,omitempty"`
								ProposerPriority string `json:"proposer_priority,omitempty"`
							} `json:"proposer,omitempty"`
							TotalVotingPower string `json:"total_voting_power,omitempty"`
						} `json:"trusted_validators,omitempty"`
					} `json:"header,omitempty"`
					Signer string `json:"signer,omitempty"`
					Packet struct {
						Sequence           string `json:"sequence,omitempty"`
						SourcePort         string `json:"source_port,omitempty"`
						SourceChannel      string `json:"source_channel,omitempty"`
						DestinationPort    string `json:"destination_port,omitempty"`
						DestinationChannel string `json:"destination_channel,omitempty"`
						Data               string `json:"data,omitempty"`
						TimeoutHeight      struct {
							RevisionNumber string `json:"revision_number,omitempty"`
							RevisionHeight string `json:"revision_height,omitempty"`
						} `json:"timeout_height,omitempty"`
						TimeoutTimestamp string `json:"timeout_timestamp,omitempty"`
					} `json:"packet,omitempty"`
					ProofCommitment string `json:"proof_commitment,omitempty"`
					ProofHeight     struct {
						RevisionNumber string `json:"revision_number,omitempty"`
						RevisionHeight string `json:"revision_height,omitempty"`
					} `json:"proof_height,omitempty"`
				} `json:"messages,omitempty"`
				Memo                        string `json:"memo,omitempty"`
				TimeoutHeight               string `json:"timeout_height,omitempty"`
				ExtensionOptions            []any  `json:"extension_options,omitempty"`
				NonCriticalExtensionOptions []any  `json:"non_critical_extension_options,omitempty"`
			} `json:"body,omitempty"`
			AuthInfo struct {
				SignerInfos []struct {
					PublicKey struct {
						Type string `json:"@type,omitempty"`
						Key  string `json:"key,omitempty"`
					} `json:"public_key,omitempty"`
					ModeInfo struct {
						Single struct {
							Mode string `json:"mode,omitempty"`
						} `json:"single,omitempty"`
					} `json:"mode_info,omitempty"`
					Sequence string `json:"sequence,omitempty"`
				} `json:"signer_infos,omitempty"`
				Fee struct {
					Amount   []any  `json:"amount,omitempty"`
					GasLimit string `json:"gas_limit,omitempty"`
					Payer    string `json:"payer,omitempty"`
					Granter  string `json:"granter,omitempty"`
				} `json:"fee,omitempty"`
			} `json:"auth_info,omitempty"`
			Signatures []string `json:"signatures,omitempty"`
		} `json:"tx,omitempty"`
		Timestamp time.Time `json:"timestamp,omitempty"`
		Events    []struct {
			Type       string `json:"type,omitempty"`
			Attributes []struct {
				Key   string `json:"key,omitempty"`
				Value any    `json:"value,omitempty"`
				Index bool   `json:"index,omitempty"`
			} `json:"attributes,omitempty"`
		} `json:"events,omitempty"`
	} `json:"txs,omitempty"`
}
