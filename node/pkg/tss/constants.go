package tss

import "time"

const (
	digestSize = 32

	notStarted uint32 = 0 // using 0 since it's the default value
	started    uint32 = 1

	// byte sizes
	hostnameSize     = 255
	pemKeySize       = 178
	trackingIDSize   = 32
	signingRoundSize = 8

	defaultMaxLiveSignatures = 1000

	defaultMaxSignerTTL = time.Minute * 5

	numBroadcastsPerSignature = 8 // GG18
	numUnicastsRounds         = 2 // GG18
)
