import { describe, jest, test, expect, it } from "@jest/globals";
import {
  GUARDIAN_VALIDATOR_VALADDR,
  HOLE_DENOM,
  TEST_WALLET_MNEMONIC_1,
  TEST_WALLET_MNEMONIC_2,
  TILTNET_GUARDIAN_PRIVATE_KEY,
  TILTNET_GUARDIAN_PUBKEY,
} from "../consts";
import { getValidators } from "../core/validator";
import {
  getAddress,
  getBalance,
  getOperatorAddress,
  getWallet,
  sendTokens,
} from "../core/walletHelpers";
import { fromBase64, toValAddress } from "wormhole-chain-sdk";
import { WormholeGuardianSet } from "wormhole-chain-sdk/lib/modules/certusone.wormholechain.wormhole/rest";

jest.setTimeout(60000);

/*
This file tests to make sure that the network can start from genesis, and then change out the guardian set.

Prerequesites: Have two nodes running - the tilt guardian validator, and the 'second' wormhole chain node.

This test will register the the public ket of the second node, and then process a governance VAA to switch over the network.

*/

test("Verify guardian validator", async () => {});

test("Process guardian set upgrade", async () => {});

test("Register guardian to validator", async () => {});
