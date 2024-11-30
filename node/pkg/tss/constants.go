package tss

import (
	"time"
	"unsafe"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
)

const (
	digestSize = 32

	notStarted uint32 = 0 // using 0 since it's the default value
	started    uint32 = 1

	// byte sizes
	hostnameSize     = 255
	pemKeySize       = 178
	signingRoundSize = 8

	// auxiliaryData is emmiterChain in bytes.
	auxiliaryDataSize = int(unsafe.Sizeof(vaa.ChainID(0)))
	maxParties        = 256

	// TrackingID = DigestSize  + auxiliaryDataSize + bitmap state of parties
	trackindIDSize = party.DigestSize + (maxParties+7)/8 + (auxiliaryDataSize)
	// *2 for hex encoding
	// 2 bytes for '-' between each field.
	trackingIDHexStrSize = 2*(trackindIDSize) + 2

	defaultMaxLiveSignatures = 1000

	defaultMaxSignerTTL     = time.Minute * 5
	defaultDelayGraceTime   = time.Second * 10
	defaultGuardianDownTime = time.Minute * 10

	numBroadcastsPerSignature = 8 // GG18
	numUnicastsRounds         = 2 // GG18

	//the assumed time thata message can be delayed between two parties.
	// for instance guardian 1 received a problem report at time 00:07, then guardian 2 can be assumed to have received the same problem report between times 00:02 and 00:12
	synchronsingInterval = time.Second * 5

	// Domain separation strings for hashing.
	// Ensures that similar digest are different for different domains.
	parsedProblemDomain = "problem"
	tssContentDomain    = "content"

	defaultMaxDownTimeJitter = time.Minute
	maxHeartbeatInterval     = defaultGuardianDownTime
)
