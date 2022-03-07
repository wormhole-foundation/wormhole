import { describe, jest, test, expect, it } from "@jest/globals";
import {
  GUARDIAN_VALIDATOR_PUBLIC_KEY,
  HOLE_DENOM,
  TEST_WALLET_MNEMONIC_1,
  TEST_WALLET_MNEMONIC_2,
} from "../consts";
import { getValidators } from "../core/validator";
import {
  faucet,
  getAddress,
  getBalance,
  getOperatorAddress,
  getWallet,
  sendTokens,
} from "../core/walletHelpers";

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
