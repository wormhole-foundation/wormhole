package guardiand

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Real API response from https://api.wormholescan.io/api/v1/observations/delegate/50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750
const testDelegateObservationsJSON = `[
  {
    "sequence": 1750,
    "emitterChain": 50,
    "emitterAddr": "00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83",
    "hash": "UqjHt/0rrOhV1iVtFII4F3cuCpRKkagyWV+b4RF1oQo=",
    "txHash": "jBQRYi+ZJKFAGdfKnMOBA68LwXfMprBnNPhDWnSYN7w=",
    "payload": "mUX/EAAAAAAAAAAAAAAAAH77OGZ111KA05quQpZKZ3beDuC9AAAAAAAAAAAAAAAAPrQYvb6VtLnPRl7PvYQkaFrNG8EAkQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABoPAAAAAAAAAAAAAAAASEtVk7u5A4P5T7KZRw8JQnz2z+IAT5lOVFQIAAAAACDebcAAAAAAAAAAAAAAAADdRood3Dktzb7222406JqjOPnxhgAAAAAAAAAAAAAAABvct/SlP0APQnP1bTAtUzDzrKSpAB4AAA==",
    "delegatedGuardianAddr": "0x000ac0076727b35fbea2dac28fee5ccb0fea768e",
    "signature": "VopDFhPNIAN6+/QcAiJ8ts9j6bzdtWNhK38uh9UdJ2JJoKScJ1makiy4fxj79aR2Wo9KF3JJqRONlUHYWUCGXAE=",
    "nonce": 0,
    "consistencyLevel": 202,
    "unreliable": false,
    "isReobservation": false,
    "verificationState": 0,
    "timestamp": "2026-04-09T23:27:36Z",
    "sentTimestamp": "2026-04-09T23:27:41Z"
  },
  {
    "sequence": 1750,
    "emitterChain": 50,
    "emitterAddr": "00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83",
    "hash": "UqjHt/0rrOhV1iVtFII4F3cuCpRKkagyWV+b4RF1oQo=",
    "txHash": "jBQRYi+ZJKFAGdfKnMOBA68LwXfMprBnNPhDWnSYN7w=",
    "payload": "mUX/EAAAAAAAAAAAAAAAAH77OGZ111KA05quQpZKZ3beDuC9AAAAAAAAAAAAAAAAPrQYvb6VtLnPRl7PvYQkaFrNG8EAkQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABoPAAAAAAAAAAAAAAAASEtVk7u5A4P5T7KZRw8JQnz2z+IAT5lOVFQIAAAAACDebcAAAAAAAAAAAAAAAADdRood3Dktzb7222406JqjOPnxhgAAAAAAAAAAAAAAABvct/SlP0APQnP1bTAtUzDzrKSpAB4AAA==",
    "delegatedGuardianAddr": "0xda798f6896a3331f64b48c12d1d57fd9cbe70811",
    "signature": "gxDWFpi3A9mG8YrZXkuHNvYzJ/GDBdkpIvvX/kQzatUW4MFscSvyF70BK2Mby5UqjZF5PKpzgtPRmlRpBEzjigA=",
    "nonce": 0,
    "consistencyLevel": 202,
    "unreliable": false,
    "isReobservation": false,
    "verificationState": 0,
    "timestamp": "2026-04-09T23:27:36Z",
    "sentTimestamp": "2026-04-09T23:27:41Z"
  },
  {
    "sequence": 1750,
    "emitterChain": 50,
    "emitterAddr": "00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83",
    "hash": "UqjHt/0rrOhV1iVtFII4F3cuCpRKkagyWV+b4RF1oQo=",
    "txHash": "jBQRYi+ZJKFAGdfKnMOBA68LwXfMprBnNPhDWnSYN7w=",
    "payload": "mUX/EAAAAAAAAAAAAAAAAH77OGZ111KA05quQpZKZ3beDuC9AAAAAAAAAAAAAAAAPrQYvb6VtLnPRl7PvYQkaFrNG8EAkQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABoPAAAAAAAAAAAAAAAASEtVk7u5A4P5T7KZRw8JQnz2z+IAT5lOVFQIAAAAACDebcAAAAAAAAAAAAAAAADdRood3Dktzb7222406JqjOPnxhgAAAAAAAAAAAAAAABvct/SlP0APQnP1bTAtUzDzrKSpAB4AAA==",
    "delegatedGuardianAddr": "0x178e21ad2e77ae06711549cfbb1f9c7a9d8096e8",
    "signature": "f5xktEwewFqribjYF0Pg2Wo9uoNjuVH/nq4a2hqenFdzEVG/g5sbRaXpCAHUR3w4ZSnUWX1Ov5ISXjU+jEwKVwE=",
    "nonce": 0,
    "consistencyLevel": 202,
    "unreliable": false,
    "isReobservation": false,
    "verificationState": 0,
    "timestamp": "2026-04-09T23:27:36Z",
    "sentTimestamp": "2026-04-09T23:27:41Z"
  },
  {
    "sequence": 1750,
    "emitterChain": 50,
    "emitterAddr": "00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83",
    "hash": "UqjHt/0rrOhV1iVtFII4F3cuCpRKkagyWV+b4RF1oQo=",
    "txHash": "jBQRYi+ZJKFAGdfKnMOBA68LwXfMprBnNPhDWnSYN7w=",
    "payload": "mUX/EAAAAAAAAAAAAAAAAH77OGZ111KA05quQpZKZ3beDuC9AAAAAAAAAAAAAAAAPrQYvb6VtLnPRl7PvYQkaFrNG8EAkQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABoPAAAAAAAAAAAAAAAASEtVk7u5A4P5T7KZRw8JQnz2z+IAT5lOVFQIAAAAACDebcAAAAAAAAAAAAAAAADdRood3Dktzb7222406JqjOPnxhgAAAAAAAAAAAAAAABvct/SlP0APQnP1bTAtUzDzrKSpAB4AAA==",
    "delegatedGuardianAddr": "0xaf45ced136b9d9e24903464ae889f5c8a723fc14",
    "signature": "jv0g6dbPsak4Gv9H3n3ZNhKU0Pq8SBBuYvSjdBtf63RbdODc2JF0xT7xr91kUrcZFZgOaCHWGhso3xz/lEhQeQE=",
    "nonce": 0,
    "consistencyLevel": 202,
    "unreliable": false,
    "isReobservation": false,
    "verificationState": 0,
    "timestamp": "2026-04-09T23:27:36Z",
    "sentTimestamp": "2026-04-09T23:27:41Z"
  },
  {
    "sequence": 1750,
    "emitterChain": 50,
    "emitterAddr": "00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83",
    "hash": "UqjHt/0rrOhV1iVtFII4F3cuCpRKkagyWV+b4RF1oQo=",
    "txHash": "jBQRYi+ZJKFAGdfKnMOBA68LwXfMprBnNPhDWnSYN7w=",
    "payload": "mUX/EAAAAAAAAAAAAAAAAH77OGZ111KA05quQpZKZ3beDuC9AAAAAAAAAAAAAAAAPrQYvb6VtLnPRl7PvYQkaFrNG8EAkQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABoPAAAAAAAAAAAAAAAASEtVk7u5A4P5T7KZRw8JQnz2z+IAT5lOVFQIAAAAACDebcAAAAAAAAAAAAAAAADdRood3Dktzb7222406JqjOPnxhgAAAAAAAAAAAAAAABvct/SlP0APQnP1bTAtUzDzrKSpAB4AAA==",
    "delegatedGuardianAddr": "0x11b39756c042441be6d8650b69b54ebe715e2343",
    "signature": "zTGBGY8fMi147mqw09gFHe5o58RJ27BKJgOeb7lwmvQb6ETBf9OYGcaJsrjfp/h8GhSX/rGrAOB7vOuMsweNVgA=",
    "nonce": 0,
    "consistencyLevel": 202,
    "unreliable": false,
    "isReobservation": false,
    "verificationState": 0,
    "timestamp": "2026-04-09T23:27:36Z",
    "sentTimestamp": "2026-04-09T23:27:40Z"
  },
  {
    "sequence": 1750,
    "emitterChain": 50,
    "emitterAddr": "00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83",
    "hash": "UqjHt/0rrOhV1iVtFII4F3cuCpRKkagyWV+b4RF1oQo=",
    "txHash": "jBQRYi+ZJKFAGdfKnMOBA68LwXfMprBnNPhDWnSYN7w=",
    "payload": "mUX/EAAAAAAAAAAAAAAAAH77OGZ111KA05quQpZKZ3beDuC9AAAAAAAAAAAAAAAAPrQYvb6VtLnPRl7PvYQkaFrNG8EAkQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAABoPAAAAAAAAAAAAAAAASEtVk7u5A4P5T7KZRw8JQnz2z+IAT5lOVFQIAAAAACDebcAAAAAAAAAAAAAAAADdRood3Dktzb7222406JqjOPnxhgAAAAAAAAAAAAAAABvct/SlP0APQnP1bTAtUzDzrKSpAB4AAA==",
    "delegatedGuardianAddr": "0x938f104aeb5581293216ce97d771e0cb721221b1",
    "signature": "z/fhKpSdTeE+hsSQs5FiYZM7azrC16NkxlvLKRZuCkso3741gsoJFydNhmEm7STzs3lrsLGDPyhAaoIK/XInjgA=",
    "nonce": 0,
    "consistencyLevel": 202,
    "unreliable": false,
    "isReobservation": false,
    "verificationState": 0,
    "timestamp": "2026-04-09T23:27:36Z",
    "sentTimestamp": "2026-04-09T23:27:40Z"
  }
]`

// TestDelegateObservationHashIsMessagePublicationHash verifies that the "hash" field returned by
// the wormholescan API for delegate observations is the Keccak256 of MessagePublication.MarshalBinary(),
// NOT the double-Keccak256 VAA body hash used by regular observations.
func TestDelegateObservationHashIsMessagePublicationHash(t *testing.T) {
	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))
	require.NotEmpty(t, observations)

	obs := observations[0]

	// Decode the hash from the API (base64).
	apiHashBytes, err := base64.StdEncoding.DecodeString(obs.MessagePublicationHash)
	require.NoError(t, err)

	// Reconstruct the MessagePublication from the observation fields.
	txHash, err := base64.StdEncoding.DecodeString(obs.TxHash)
	require.NoError(t, err)
	payload, err := base64.StdEncoding.DecodeString(obs.Payload)
	require.NoError(t, err)
	emitterAddr, err := vaa.BytesToAddress(mustDecodeHex(t, obs.EmitterAddr))
	require.NoError(t, err)
	ts, err := time.Parse(time.RFC3339, obs.Timestamp)
	require.NoError(t, err)

	mp := &node_common.MessagePublication{
		TxID:             txHash,
		Timestamp:        ts,
		Nonce:            obs.Nonce,
		Sequence:         obs.Sequence,
		ConsistencyLevel: obs.ConsistencyLevel,
		EmitterChain:     vaa.ChainID(obs.EmitterChain),
		EmitterAddress:   emitterAddr,
		Payload:          payload,
		IsReobservation:  obs.IsReobservation,
		Unreliable:       obs.Unreliable,
	}

	buf, err := mp.MarshalBinary()
	require.NoError(t, err)

	mpHash := crypto.Keccak256Hash(buf)
	assert.Equal(t, apiHashBytes, mpHash.Bytes(),
		"API hash should match Keccak256(MessagePublication.MarshalBinary())")

	// Also compute the VAA body hash (double Keccak256) using the SDK and confirm it does NOT match.
	v := &vaa.VAA{
		Timestamp:        ts,
		Nonce:            obs.Nonce,
		EmitterChain:     vaa.ChainID(obs.EmitterChain),
		EmitterAddress:   emitterAddr,
		Sequence:         obs.Sequence,
		ConsistencyLevel: obs.ConsistencyLevel,
		Payload:          payload,
	}
	vaaHash := v.SigningDigest()
	assert.NotEqual(t, apiHashBytes, vaaHash.Bytes(),
		"API hash should NOT match the double-Keccak256 VAA body hash")
}

func mustDecodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestDelegateObservationVAAHash(t *testing.T) {
	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))
	require.NotEmpty(t, observations)

	hash, err := delegateObservationVAAHash(&observations[0])
	require.NoError(t, err)

	// Known VAA hash for 50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750
	// https://api.wormholescan.io/api/v1/observations/50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750
	expectedHashBytes, err := base64.StdEncoding.DecodeString("QY3pqI+RYqV1v1yfFK3CvJ1swgaitqJStCIr/Sb70MM=")
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("0x%x", expectedHashBytes), hash)

	// All observations in the test data share the same VAA body, so they should all produce the same hash.
	for i := range observations {
		h, err := delegateObservationVAAHash(&observations[i])
		require.NoError(t, err)
		assert.Equal(t, hash, h, "observation %d should have the same VAA hash", i)
	}
}

func TestBuildDelegateSignaturesBroadcasts(t *testing.T) {
	vaaID := "50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750"

	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

	broadcasts, err := buildDelegateSignaturesBroadcasts(vaaID, observations)
	require.NoError(t, err)
	// All test observations share the same MessagePublicationHash, so we get one broadcast.
	require.Len(t, broadcasts, 1)
	broadcast := broadcasts[0]

	// All 6 signatures should pass verification.
	assert.Equal(t, 6, len(broadcast.Signatures))

	// Verify common fields from the API response.
	assert.Equal(t, uint32(50), broadcast.EmitterChain)
	assert.Equal(t, uint64(1750), broadcast.Sequence)
	assert.Equal(t, uint32(0), broadcast.Nonce)
	assert.Equal(t, uint32(202), broadcast.ConsistencyLevel)
	assert.False(t, broadcast.Unreliable)
	assert.False(t, broadcast.IsReobservation)
	assert.Equal(t, uint32(0), broadcast.VerificationState)

	// Verify each signature has the expected guardian address.
	expectedGuardians := map[string]bool{
		"000ac0076727b35fbea2dac28fee5ccb0fea768e": false,
		"da798f6896a3331f64b48c12d1d57fd9cbe70811": false,
		"178e21ad2e77ae06711549cfbb1f9c7a9d8096e8": false,
		"af45ced136b9d9e24903464ae889f5c8a723fc14": false,
		"11b39756c042441be6d8650b69b54ebe715e2343": false,
		"938f104aeb5581293216ce97d771e0cb721221b1": false,
	}
	for _, sig := range broadcast.Signatures {
		addr := fmt.Sprintf("%x", sig.GuardianAddr)
		_, ok := expectedGuardians[addr]
		assert.True(t, ok, "unexpected guardian address: %s", addr)
		expectedGuardians[addr] = true
		assert.NotEmpty(t, sig.Signature)
		assert.NotZero(t, sig.SentTimestamp)
	}
	for addr, seen := range expectedGuardians {
		assert.True(t, seen, "missing guardian address: %s", addr)
	}
}

func TestBuildDelegateSignaturesBroadcasts_BadSignatureLength(t *testing.T) {
	// Signature decodes to 66 bytes (not 65) — should be dropped by length check.
	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

	observations[0].Signature = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	vaaID := "50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750"
	broadcasts, err := buildDelegateSignaturesBroadcasts(vaaID, observations)
	require.NoError(t, err)
	require.Len(t, broadcasts, 1)

	// 5 of 6 should pass (the wrong-length one is dropped).
	assert.Equal(t, 5, len(broadcasts[0].Signatures))
}

func TestBuildDelegateSignaturesBroadcasts_BadSignatureEcrecover(t *testing.T) {
	// Signature is 65 bytes but wrong — should be dropped by ecrecover/address mismatch.
	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

	observations[0].Signature = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

	vaaID := "50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750"
	broadcasts, err := buildDelegateSignaturesBroadcasts(vaaID, observations)
	require.NoError(t, err)
	require.Len(t, broadcasts, 1)

	// 5 of 6 should pass (the corrupted one is dropped).
	assert.Equal(t, 5, len(broadcasts[0].Signatures))
}

// TestBuildDelegateSignaturesBroadcasts_NonVAAFieldsAreSignaturePreimage verifies that
// non-VAA fields (txID, unreliable, isReobservation, verificationState) are part of the delegate
// observation signature preimage. Changing any of these fields should cause signature
// verification to fail, dropping all signatures.
func TestBuildDelegateSignaturesBroadcasts_NonVAAFieldsAreSignaturePreimage(t *testing.T) {
	vaaID := "50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750"

	tests := []struct {
		name   string
		mutate func(obs []wormholescanDelegateObservation)
	}{
		{
			name: "txID changed",
			mutate: func(obs []wormholescanDelegateObservation) {
				for i := range obs {
					obs[i].TxHash = base64.StdEncoding.EncodeToString([]byte("different_tx_hash_value_here!"))
				}
			},
		},
		{
			name: "unreliable set to true",
			mutate: func(obs []wormholescanDelegateObservation) {
				for i := range obs {
					obs[i].Unreliable = true
				}
			},
		},
		{
			name: "isReobservation set to true",
			mutate: func(obs []wormholescanDelegateObservation) {
				for i := range obs {
					obs[i].IsReobservation = true
				}
			},
		},
		{
			name: "verificationState set to non-zero",
			mutate: func(obs []wormholescanDelegateObservation) {
				for i := range obs {
					obs[i].VerificationState = 1
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var observations []wormholescanDelegateObservation
			require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

			tc.mutate(observations)

			// All signatures should fail verification because the preimage changed.
			_, err := buildDelegateSignaturesBroadcasts(vaaID, observations)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no signatures passed verification")
		})
	}
}

// TestBuildDelegateSignaturesBroadcasts_MixedNonVAAFields verifies that observations with
// mixed non-VAA field values (e.g., some with unreliable=true and some with unreliable=false)
// are handled correctly — only the signatures that match the actual signed preimage survive
// verification.
func TestBuildDelegateSignaturesBroadcasts_MixedNonVAAFields(t *testing.T) {
	vaaID := "50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750"

	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))
	require.True(t, len(observations) >= 2, "need at least 2 observations for this test")

	// Mutate half the observations to have different non-VAA fields.
	// The real observations were signed with unreliable=false, so flipping half should
	// cause those to fail verification while the rest still pass.
	for i := range observations {
		if i%2 == 0 {
			observations[i].Unreliable = true
		}
	}

	broadcasts, err := buildDelegateSignaturesBroadcasts(vaaID, observations)
	require.NoError(t, err)
	require.NotEmpty(t, broadcasts)

	// Only the unmodified observations (odd indices) should have valid signatures.
	totalSigs := 0
	for _, b := range broadcasts {
		totalSigs += len(b.Signatures)
	}
	assert.Equal(t, 3, totalSigs, "only half the signatures should survive verification")
}

func TestBuildDelegateSignaturesBroadcasts_EmptyObservations(t *testing.T) {
	_, err := buildDelegateSignaturesBroadcasts("50/abc/123", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no delegate observations provided")
}

func TestBuildDelegateSignaturesBroadcasts_BadVaaID(t *testing.T) {
	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

	_, err := buildDelegateSignaturesBroadcasts("invalid", observations)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vaa_id must be in format")
}
