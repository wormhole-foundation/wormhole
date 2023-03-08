package guardiand

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/status-im/keycard-go/hexutils"
)

var AdminClientGovernanceVAAVerifyCmd = &cobra.Command{
	Use:   "governance-vaa-verify [FILENAME]",
	Short: "Verify governance vaa in prototxt format (offline)",
	Run:   runGovernanceVAAVerify,
	Args:  cobra.ExactArgs(1),
}

func runGovernanceVAAVerify(cmd *cobra.Command, args []string) {
	path := args[0]

	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	var req nodev1.InjectGovernanceVAARequest
	err = prototext.Unmarshal(b, &req)
	if err != nil {
		log.Fatalf("failed to deserialize: %v", err)
	}

	timestamp := time.Unix(int64(req.Timestamp), 0)

	for _, message := range req.Messages {
		var (
			v *vaa.VAA
		)
		switch payload := message.Payload.(type) {
		case *nodev1.GovernanceMessage_GuardianSet:
			v, err = adminGuardianSetUpdateToVAA(payload.GuardianSet, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_ContractUpgrade:
			v, err = adminContractUpgradeToVAA(payload.ContractUpgrade, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_BridgeRegisterChain:
			v, err = tokenBridgeRegisterChain(payload.BridgeRegisterChain, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_BridgeContractUpgrade:
			v, err = tokenBridgeUpgradeContract(payload.BridgeContractUpgrade, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_AccountantModifyBalance:
			v, err = accountantModifyBalance(payload.AccountantModifyBalance, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_WormchainStoreCode:
			v, err = wormchainStoreCode(payload.WormchainStoreCode, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_WormchainInstantiateContract:
			v, err = wormchainInstantiateContract(payload.WormchainInstantiateContract, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_WormchainMigrateContract:
			v, err = wormchainMigrateContract(payload.WormchainMigrateContract, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_CircleIntegrationUpdateWormholeFinality:
			v, err = circleIntegrationUpdateWormholeFinality(payload.CircleIntegrationUpdateWormholeFinality, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_CircleIntegrationRegisterEmitterAndDomain:
			v, err = circleIntegrationRegisterEmitterAndDomain(payload.CircleIntegrationRegisterEmitterAndDomain, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_CircleIntegrationUpgradeContractImplementation:
			v, err = circleIntegrationUpgradeContractImplementation(payload.CircleIntegrationUpgradeContractImplementation, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		case *nodev1.GovernanceMessage_WormholeRelayerSetDefaultRelayProvider:
			v, err = wormholeRelayerSetDefaultRelayProvider(payload.WormholeRelayerSetDefaultRelayProvider, timestamp, req.CurrentSetIndex, message.Nonce, message.Sequence)
		default:
			panic(fmt.Sprintf("unsupported VAA type: %T", payload))
		}
		if err != nil {
			log.Fatalf("invalid update: %v", err)
		}

		digest := v.SigningDigest().Bytes()
		if err != nil {
			panic(err)
		}

		b, err := v.Marshal()
		if err != nil {
			panic(err)
		}

		log.Printf("Serialized: %v", hex.EncodeToString(b))

		log.Printf("VAA with digest %s: %+v", hexutils.BytesToHex(digest), spew.Sdump(v))
	}
}
