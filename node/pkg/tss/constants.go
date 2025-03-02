package tss

import (
	"time"
	"unsafe"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/tss-lib/v2/ecdsa/keygen"
	"github.com/xlabs/tss-lib/v2/ecdsa/party"
	"github.com/xlabs/tss-lib/v2/ecdsa/resharing"
	"github.com/xlabs/tss-lib/v2/ecdsa/signing"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var tssProtoMessageNames = []string{}

var tssProtoMessageSize = 0

func init() {
	tssProtoMessageNames = append(tssProtoMessageNames, extractProtoTypeNames(signing.File_protob_ecdsa_signing_proto)...)
	tssProtoMessageNames = append(tssProtoMessageNames, extractProtoTypeNames(keygen.File_protob_ecdsa_keygen_proto)...)
	tssProtoMessageNames = append(tssProtoMessageNames, extractProtoTypeNames(resharing.File_protob_ecdsa_resharing_proto)...)

	for _, name := range tssProtoMessageNames {
		tssProtoMessageSize = max(tssProtoMessageSize, len(name))
	}
}

func extractProtoTypeNames(protoreflectDesc protoreflect.FileDescriptor) []string {
	names := make([]string, protoreflectDesc.Messages().Len())

	for i := range protoreflectDesc.Messages().Len() {
		m := protoreflectDesc.Messages().Get(i)
		names[i] = string(m.FullName())
	}

	return names
}

const (
	DefaultPort = "8998"

	digestSize = 32

	notStarted uint32 = 0 // using 0 since it's the default value
	started    uint32 = 1

	// byte sizes
	hostnameSize = 255
	pemKeySize   = 178

	// auxiliaryData is emmiterChain in bytes.
	auxiliaryDataSize = int(unsafe.Sizeof(vaa.ChainID(0)))
	maxParties        = 256

	// TrackingID = DigestSize  + auxiliaryDataSize + bitmap state of parties
	trackindIDSize = party.DigestSize + (maxParties+7)/8 + (auxiliaryDataSize)
	// *2 for hex encoding
	// 2 bytes for '-' between each field.
	trackingIDHexStrSize = 2*(trackindIDSize) + 2

	defaultMaxLiveSignatures = 20000

	// Since each sigState is created via almost any of the ftCommands, I decided on setting it as 1000 sigs a minute
	// and multiplied it by number of minutes we have
	sigStateRateLimit = defaultMaxLiveSignatures * int(2*defaultMaxSignerTTL/time.Minute)

	defaultMaxSignerTTL     = time.Minute * 5
	defaultDelayGraceTime   = time.Minute
	defaultGuardianDownTime = time.Minute * 10

	numBroadcastsPerSignature = 8 // GG18
	numUnicastsRounds         = 2 // GG18

	// the assumed time that a message can be delayed between two parties.
	// for instance guardian 1 received a problem report at time 00:07, then guardian 2 can be
	// assumed to have received the same problem report between times 00:02 and 00:12
	synchronsingInterval = time.Second * 5

	// Domain separation strings for hashing.
	// Ensures that similar digest are different for different domains.
	parsedProblemDomain  = "problem"
	tssContentDomain     = "content"
	newAnouncementDomain = "anncmnt"

	defaultMaxDownTimeJitter = time.Minute
	maxHeartbeatInterval     = defaultGuardianDownTime

	// Consistency levels (following https://wormhole.com/docs/build/reference/consistency-levels/):
	instantConsistencyLevel uint8 = vaa.ConsistencyLevelPublishImmediately // low consistancy

	pythnetFinalizedConsistencyLevel uint8 = 1
	solanaFinalizedConsistencyLevel  uint8 = 1

	senderIndexSize = int(unsafe.Sizeof(senderIndex(0)))
)
