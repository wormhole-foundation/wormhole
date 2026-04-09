package guardiand

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestBuildDelegateSignaturesBroadcast(t *testing.T) {
	vaaID := "50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750"

	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

	broadcast, err := buildDelegateSignaturesBroadcast(vaaID, observations)
	require.NoError(t, err)
	require.NotNil(t, broadcast)

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

func TestBuildDelegateSignaturesBroadcast_BadSignature(t *testing.T) {
	// Take the real data but corrupt one signature — it should be dropped.
	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

	observations[0].Signature = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"

	vaaID := "50/00000000000000000000000062deeafee06c7442a21c93ededc79a0cb5791c83/1750"
	broadcast, err := buildDelegateSignaturesBroadcast(vaaID, observations)
	require.NoError(t, err)
	require.NotNil(t, broadcast)

	// 5 of 6 should pass (the corrupted one is dropped).
	assert.Equal(t, 5, len(broadcast.Signatures))
}

func TestBuildDelegateSignaturesBroadcast_EmptyObservations(t *testing.T) {
	_, err := buildDelegateSignaturesBroadcast("50/abc/123", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no delegate observations provided")
}

func TestBuildDelegateSignaturesBroadcast_BadVaaID(t *testing.T) {
	var observations []wormholescanDelegateObservation
	require.NoError(t, json.Unmarshal([]byte(testDelegateObservationsJSON), &observations))

	_, err := buildDelegateSignaturesBroadcast("invalid", observations)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vaa_id must be in format")
}
