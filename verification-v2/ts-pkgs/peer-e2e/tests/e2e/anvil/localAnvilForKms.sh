#!/bin/bash

set -meuo pipefail

# Anvil Private Key
PRIVATE_KEY=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80

# Contract Addresses
MOCK_ADDRESS=0x5FbDB2315678afecb367f032d93F642f64180aa3
VERIFIER_ADDRESS=0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512

# Guardian Set (single KMS guardian)
APPEND_SET_FUNCTION_SIG="appendGuardianSet((address[], uint32))"
EXPIRATION_TIME=$(date -d "1 year" +%s)

GUARDIAN_ADDRESSES=(
  "0x9197313AA3c2004e7C4E66B7243C86Df117B55a4"
)

IFS=,
GUARDIAN_SET="([${GUARDIAN_ADDRESSES[*]}], $EXPIRATION_TIME)"

# Verifier Update
UPDATE_FUNCTION_SIG="update(bytes)"
PULL_MESSAGE=0x0200000001

anvil --quiet --host 0.0.0.0 &

deadline=$((SECONDS+60))
until cast block-number >/dev/null 2>&1; do
    if [ "$SECONDS" -ge "$deadline" ]; then
        echo "Timed out waiting for anvil" >&2
        exit 1
    fi
    sleep 0.5
done

forge create --private-key "$PRIVATE_KEY" --broadcast test/WormholeVerifier.t.sol:WormholeV1Mock
forge create WormholeVerifier --private-key "$PRIVATE_KEY" --broadcast --constructor-args $MOCK_ADDRESS 0 0 0 0x
cast send --private-key "$PRIVATE_KEY" "$MOCK_ADDRESS" "$APPEND_SET_FUNCTION_SIG" "$GUARDIAN_SET"
cast send --private-key "$PRIVATE_KEY" "$VERIFIER_ADDRESS" "$UPDATE_FUNCTION_SIG" "$PULL_MESSAGE"
fg

