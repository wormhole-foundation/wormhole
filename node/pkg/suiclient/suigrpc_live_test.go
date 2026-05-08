package suiclient

import (
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// To enable live tests, set the environmental variable SUI_GRPC_TEST_LIVE=ANYTHING.
// Set it to INCLUDE_SUBSCRIPTION_TEST specifically to enable the subscription test.
func suiGrpcClientLiveTestEnabled() (bool, bool) {
	val := os.Getenv("SUI_GRPC_TEST_LIVE")

	runNormalTests := false
	runSubscriptionTest := false

	if len(val) > 0 {
		runNormalTests = true
	}

	if val == "INCLUDE_SUBSCRIPTION_TEST" {
		runSubscriptionTest = true
	}

	return runNormalTests, runSubscriptionTest
}

// Definition converted from the Sui module's definition.
type WormholeState struct {
	ID              [32]byte // UID is represented in Go as a 32-byte list
	GovernanceChain uint16
}

func TestGrpcClientGetObject(t *testing.T) {
	runNormalTests, _ := suiGrpcClientLiveTestEnabled()
	if !runNormalTests {
		return
	}

	testLogger := zap.NewNop()
	ctx := context.Background()
	client, err := NewSuiGrpcClient(SuiRPCMainnet, testLogger)
	require.NoError(t, err)

	// Sample Object ID: https://suivision.xyz/object/0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9
	objectId := "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9"
	obj, err := client.GetObject(ctx, objectId)

	require.NoError(t, err)

	state := DecodeBcs[WormholeState](obj.BcsBytes)
	require.NotNil(t, state)

	fmt.Printf("[+] Object %s:\n", obj.ID)
	fmt.Println("\tObjectType: ", obj.ObjectType)
	fmt.Println("\tBcsType: ", obj.BcsType)
	fmt.Println("\tBcsBytes: ", hex.EncodeToString(obj.BcsBytes))
	fmt.Println("\tState: ")
	fmt.Println("\t\tID: ", hex.EncodeToString(state.ID[:]))
	fmt.Println("\t\tGovernanceChain: ", state.GovernanceChain)

}

// Definition converted from the Sui module's definition.
type WormholeMessage struct {
	Sender           [32]byte
	Sequence         uint64
	Nonce            uint32
	Payload          []byte
	ConsistencyLevel uint8
	Timestamp        uint64
}

func TestGrpcClientGetTransaction(t *testing.T) {
	runNormalTests, _ := suiGrpcClientLiveTestEnabled()
	if !runNormalTests {
		return
	}

	testLogger := zap.NewNop()
	ctx := context.Background()
	client, err := NewSuiGrpcClient(SuiRPCMainnet, testLogger)
	require.NoError(t, err)

	// Sample transaction block: https://suivision.xyz/txblock/55q6tfVjenQwG8Uyn5EDQBuAjZHbPqdDaB9t8qAhYgBi
	transactionDigest := "HUcA9av3dfKgGmzZ8eiw157824Y1nBSEfmUHbo4fs9dS"
	tx, err := client.GetTransaction(ctx, transactionDigest)

	require.NoError(t, err)

	fmt.Println("[+] Transaction Digest: ", tx.Digest)
	for _, ev := range tx.Events {
		fmt.Printf("\tEventType=%s\n", ev.EventType)
		fmt.Println("\t\tPackageID:", ev.PackageID)
		fmt.Println("\t\tModule:", ev.TransactionModule)
		fmt.Println("\t\tSender: ", ev.Sender)
		fmt.Println("\t\tBcsType: ", ev.BcsType)
		fmt.Println("\t\tBcsBytes: ", hex.EncodeToString(ev.BcsBytes))

		if ev.EventType == "0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage" {
			m := DecodeBcs[WormholeMessage](ev.BcsBytes)

			require.NotNil(t, m)

			fmt.Println("\t\tMessage Publication:")
			fmt.Println("\t\t\tSender: ", hex.EncodeToString(m.Sender[:]))
			fmt.Println("\t\t\tSequence: ", m.Sequence)
			fmt.Println("\t\t\tNonce: ", m.Nonce)
			fmt.Println("\t\t\tPayload: ", hex.EncodeToString(m.Payload))
			fmt.Println("\t\t\tConsistencyLevel: ", m.ConsistencyLevel)
			fmt.Println("\t\t\tTimestamp: ", m.Timestamp)
		}
	}
}

func TestGrpcClientGetCheckpointSN(t *testing.T) {
	runNormalTests, _ := suiGrpcClientLiveTestEnabled()
	if !runNormalTests {
		return
	}

	testLogger := zap.NewNop()
	ctx := context.Background()
	client, err := NewSuiGrpcClient(SuiRPCMainnet, testLogger)
	require.NoError(t, err)

	sequenceNumber, err := client.GetLatestCheckpointSN(ctx)
	require.NoError(t, err)
	fmt.Println("Sequence Number: ", sequenceNumber)
}

func TestGrpcClientSubscribeToEvents(t *testing.T) {
	_, runSubscriptionTest := suiGrpcClientLiveTestEnabled()
	if !runSubscriptionTest {
		return
	}

	testLogger := zap.NewNop()
	ctx := context.Background()
	client, err := NewSuiGrpcClient(SuiRPCMainnet, testLogger)
	require.NoError(t, err)

	eventTypes := []string{
		"0x2c8d603bc51326b8c13cef9dd07031a408a48dddb541963357661df5d3204809::order_info::OrderPlaced",
		"0x5306f64e312b581766351c07af79c72fcb1cd25147157fdc2f8ad76de9a3fb6a::publish_message::WormholeMessage",
	}

	suiEventChan := make(chan SuiEvent)
	subscription, err := client.SubscribeToEvents(ctx, eventTypes, suiEventChan)
	require.NoError(t, err)
	defer subscription.Unsubscribe()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			{
				fmt.Println("context done. breaking.")
				return
			}
		case subErr := <-subscription.Err():
			{
				fmt.Printf("subscription reported an error: %v\n", subErr)
			}
		case <-ticker.C:
			{
				fmt.Println("calling unsubscribe")
				subscription.Unsubscribe()
				time.Sleep(2 * time.Second) //nolint:forbidigo // Using time.Sleep here as an arbitrary delay to ensure the subscription closes internally
				return
			}
		case ev := <-suiEventChan:
			{
				fmt.Printf("\tEventType=%s\n", ev.EventType)
				fmt.Println("\t\tPackageID:", ev.PackageID)
				fmt.Println("\t\tModule:", ev.TransactionModule)
				fmt.Println("\t\tSender: ", ev.Sender)
				fmt.Println("\t\tBcsType: ", ev.BcsType)
				fmt.Println("\t\tBcsBytes: ", hex.EncodeToString(ev.BcsBytes))
			}
		}
	}
}
