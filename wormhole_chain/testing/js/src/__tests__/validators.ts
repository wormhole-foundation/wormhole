import { describe, jest, test, expect, it } from "@jest/globals";
import {
  GUARDIAN_VALIDATOR_PUBLIC_KEY,
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
import {
  executeGovernanceVAA,
  getGuardianSets,
  getGuardianValidatorRegistrations,
  fromBase64,
  toValAddress,
  registerGuardianValidator,
} from "wormhole-chain-sdk";
import { WormholeGuardianSet } from "wormhole-chain-sdk/lib/modules/certusone.wormholechain.wormhole/rest";

jest.setTimeout(60000);

/*
This file tests to make sure that the network can start from genesis, and then change out the guardian set.

Prerequesites: Have two nodes running - the tilt guardian validator, and the 'second' wormhole chain node.

This test will register the the public ket of the second node, and then process a governance VAA to switch over the network.

*/

test("Verify guardian validator", async () => {
  const validators = await getValidators();
  const guardianValidator = validators.validators.find(
    (x) => x.operatorAddress === GUARDIAN_VALIDATOR_PUBLIC_KEY
  );

  console.log(
    "OPERATOR ADDRESS",
    await getOperatorAddress(TEST_WALLET_MNEMONIC_1)
  );

  console.log("VALIDATORS", validators);
  expect(!!guardianValidator == true).toBe(true);

  //TODO determine if active validators should be instantly bonded / unbonded. What impact does this have on delegated bonds?
  //Alternately, if bonded !== active, how will validators become bonded?
  expect(guardianValidator?.status).toBe(3); //BondStatus can't be easily imported, but 3 evidently means bonded.

  //Pull our guardian set
  const guardianSets = await getGuardianSets();
  expect(guardianSets.GuardianSet?.length).toBe(1);

  const guardianSet: WormholeGuardianSet[] = guardianSets.GuardianSet || [];
  const foundTiltGuardian = guardianSet[0].keys?.find(
    (x) => x === TILTNET_GUARDIAN_PUBKEY
  );

  expect(!!foundTiltGuardian).toBe(true);

  const guardianRegistrations = await getGuardianValidatorRegistrations();
  const guardianValidators = guardianRegistrations.guardianValidator || [];
  const foundGuardianValidator = guardianValidators.find(
    (x) =>
      x.validatorAddr &&
      x.guardianKey &&
      x.guardianKey === TILTNET_GUARDIAN_PUBKEY &&
      toValAddress(fromBase64(x.validatorAddr)) ===
        GUARDIAN_VALIDATOR_PUBLIC_KEY
  );

  expect(!!foundGuardianValidator).toBe(true);
});

//TODO sequence number mismatch when rerunning tests?
test("Process guardian set upgrade", async () => {
  const wallet1 = await getWallet(TEST_WALLET_MNEMONIC_1);
  const result = await executeGovernanceVAA(wallet1, "not a real thing");

  console.log("RESULT FROM GUARDIAN SET UPGRADE", result);
});

test("Register guardian to validator", async () => {
  const wallet1 = await getWallet(TEST_WALLET_MNEMONIC_1);
  const result = await registerGuardianValidator(
    wallet1,
    TILTNET_GUARDIAN_PUBKEY,
    TILTNET_GUARDIAN_PRIVATE_KEY,
    GUARDIAN_VALIDATOR_PUBLIC_KEY
  );

  console.log("RESULT FROM REGISTER GUARDIAN VALIDATOR", result);
});

test("Full bootstrap test", async () => {
  //verify that guardian 1 is the only active guardian.
  //verify that guardian 1 is registered to test wallet 1.
  //verify that guardian 1 is bonded.
  //verify that guardian 2 is not active
  //verify that guardian 2 does not have any registrations
  //verify that guardian 2 is not bonded.
  //verify that the guardian set is 1
  //verify that the only guardian public key is guardian public key 1.
  //register guardian 2 to wallet 2.
  //verify that guardian 2 is registered to wallet 2.
  //verify that guardian 2 is not bonded
  //verify that guardian 2 is not active
  //process upgrade vaa
  //verify that the guardian set is 2
  //verify that the only guardian public key is guardian public key 2.
  //verify that guardian 2 is bonded
  //verify that guardain 2 is active
  //verify that guardian 1 is not bonded
  //verify that guardian 1 is not active
});

//TODO delegator tests
